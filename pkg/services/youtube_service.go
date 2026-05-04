package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go-secrets-pipeline/pkg/config"
	"go-secrets-pipeline/pkg/models"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type YouTubeService struct {
	cfg *config.Config
}

type YouTubeUploadRequest struct {
	Script    *models.Script
	Result    *models.ProductionResult
	PublishAt time.Time
}

type YouTubeUploadResult struct {
	VideoID         string
	VideoURL        string
	FileName        string
	PublishAt       time.Time
	PlaylistWarning string
}

type YouTubeChannelInfo struct {
	ID        string
	Title     string
	CustomURL string
}

func NewYouTubeService(cfg *config.Config) *YouTubeService {
	return &YouTubeService{cfg: cfg}
}

func (s *YouTubeService) Authorize(ctx context.Context) (string, []YouTubeChannelInfo, error) {
	if strings.TrimSpace(s.cfg.YouTubeClientID) == "" {
		return "", nil, fmt.Errorf("YOUTUBE_CLIENT_ID is required")
	}
	if strings.TrimSpace(s.cfg.YouTubeClientSecret) == "" {
		return "", nil, fmt.Errorf("YOUTUBE_CLIENT_SECRET is required")
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	server := &http.Server{}

	mux := http.NewServeMux()
	listener, err := listenLocalhost()
	if err != nil {
		return "", nil, err
	}
	redirectURL := "http://" + listener.Addr().String() + "/oauth2callback"

	conf := s.oauthConfig(redirectURL)
	mux.HandleFunc("/oauth2callback", func(w http.ResponseWriter, r *http.Request) {
		if errText := r.URL.Query().Get("error"); errText != "" {
			errCh <- fmt.Errorf("oauth error: %s", errText)
			http.Error(w, errText, http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("oauth code is empty")
			http.Error(w, "oauth code is empty", http.StatusBadRequest)
			return
		}
		fmt.Fprintln(w, "OAuth готов. Можно вернуться в терминал.")
		codeCh <- code
	})
	server.Handler = mux
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer server.Shutdown(ctx)

	authURL := conf.AuthCodeURL(
		"youtube-auth",
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)
	fmt.Printf("Открой URL и выбери аккаунт/канал YouTube:\n%s\n\n", authURL)
	_ = exec.Command("open", authURL).Start()

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return "", nil, err
	case <-ctx.Done():
		return "", nil, ctx.Err()
	}

	token, err := conf.Exchange(ctx, code)
	if err != nil {
		return "", nil, fmt.Errorf("oauth exchange: %w", err)
	}
	if token.RefreshToken == "" {
		return "", nil, fmt.Errorf("Google не вернул refresh_token; удали доступ приложения в myaccount.google.com/permissions и повтори")
	}

	channels, err := s.channelsForClient(ctx, conf.Client(ctx, token))
	if err != nil {
		return "", nil, err
	}
	return token.RefreshToken, channels, nil
}

func (s *YouTubeService) Upload(ctx context.Context, req YouTubeUploadRequest) (*YouTubeUploadResult, error) {
	if err := s.validateConfig(); err != nil {
		return nil, err
	}
	if req.Script == nil || req.Result == nil {
		return nil, fmt.Errorf("script and result are required")
	}

	client := s.oauthClient(ctx)
	yt, err := youtube.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("youtube service: %w", err)
	}

	file, err := os.Open(req.Result.VideoPath)
	if err != nil {
		return nil, fmt.Errorf("open video: %w", err)
	}
	defer file.Close()

	video := &youtube.Video{
		Snippet: &youtube.VideoSnippet{
			Title:       req.Script.Title,
			Description: req.Script.NarrationText,
			CategoryId:  "27",
		},
		Status: &youtube.VideoStatus{
			PrivacyStatus:           "private",
			PublishAt:               req.PublishAt.UTC().Format(time.RFC3339),
			SelfDeclaredMadeForKids: false,
		},
	}

	fileName := uploadFileName(req)
	insert := yt.Videos.Insert([]string{"snippet", "status"}, video).Media(file)
	insert.Header().Set("Slug", fileName)

	created, err := insert.Do()
	if err != nil {
		return nil, fmt.Errorf("youtube upload: %w", err)
	}

	playlistID := strings.TrimSpace(s.cfg.YouTubePlaylistID)
	if err := s.addToPlaylist(ctx, yt, playlistID, created.Id); err != nil {
		return &YouTubeUploadResult{
			VideoID:         created.Id,
			VideoURL:        "https://www.youtube.com/watch?v=" + created.Id,
			FileName:        fileName,
			PublishAt:       req.PublishAt,
			PlaylistWarning: err.Error(),
		}, nil
	}

	return &YouTubeUploadResult{
		VideoID:   created.Id,
		VideoURL:  "https://www.youtube.com/watch?v=" + created.Id,
		FileName:  fileName,
		PublishAt: req.PublishAt,
	}, nil
}

func uploadFileName(req YouTubeUploadRequest) string {
	if req.Result.VideoPath != "" {
		if base := filepath.Base(req.Result.VideoPath); base != "." && base != string(filepath.Separator) {
			return base
		}
	}
	return "video.mp4"
}

func (s *YouTubeService) validateConfig() error {
	if strings.TrimSpace(s.cfg.YouTubeClientID) == "" {
		return fmt.Errorf("YOUTUBE_CLIENT_ID is required")
	}
	if strings.TrimSpace(s.cfg.YouTubeClientSecret) == "" {
		return fmt.Errorf("YOUTUBE_CLIENT_SECRET is required")
	}
	if strings.TrimSpace(s.cfg.YouTubeRefreshToken) == "" {
		return fmt.Errorf("YOUTUBE_REFRESH_TOKEN is required")
	}
	return nil
}

func (s *YouTubeService) oauthClient(ctx context.Context) *http.Client {
	conf := s.oauthConfig("")
	token := &oauth2.Token{RefreshToken: s.cfg.YouTubeRefreshToken}
	return conf.Client(ctx, token)
}

func (s *YouTubeService) oauthConfig(redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.cfg.YouTubeClientID,
		ClientSecret: s.cfg.YouTubeClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURL,
		Scopes:       []string{youtube.YoutubeScope},
	}
}

func (s *YouTubeService) addToPlaylist(ctx context.Context, yt *youtube.Service, playlistID, videoID string) error {
	item := &youtube.PlaylistItem{
		Snippet: &youtube.PlaylistItemSnippet{
			PlaylistId: playlistID,
			ResourceId: &youtube.ResourceId{
				Kind:    "youtube#video",
				VideoId: videoID,
			},
		},
	}
	if _, err := yt.PlaylistItems.Insert([]string{"snippet"}, item).Context(ctx).Do(); err != nil {
		return fmt.Errorf("youtube playlist insert: %w", err)
	}
	return nil
}

func (s *YouTubeService) channelsForClient(ctx context.Context, client *http.Client) ([]YouTubeChannelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/youtube/v3/channels?part=snippet&mine=true", nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("youtube channels list: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("youtube channels list %d: %s", resp.StatusCode, string(body))
	}

	var raw struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title     string `json:"title"`
				CustomURL string `json:"customUrl"`
			} `json:"snippet"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	channels := make([]YouTubeChannelInfo, 0, len(raw.Items))
	for _, item := range raw.Items {
		channels = append(channels, YouTubeChannelInfo{
			ID:        item.ID,
			Title:     item.Snippet.Title,
			CustomURL: item.Snippet.CustomURL,
		})
	}
	return channels, nil
}

func listenLocalhost() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}
