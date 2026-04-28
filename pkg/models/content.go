package models

// RawContent — содержимое распарсенного markdown-файла
type RawContent struct {
	FilePath    string
	FileNum     int    // номер из имени файла (line-NNN)
	Title       string // первая строка или имя файла
	Explanation string // основной текст объяснения
	CodeBlocks  []CodeBlock
	OldBlocks   []string // блоки ```old
}

type CodeBlock struct {
	Lang string // "go", "mermaid", "bash", etc.
	Code string
}
