Секрет в том, что при использовании `go run` исходный код сначала компилируется и запускается из временного каталога внутри `go-build`, а при `go build` создается бинарник в текущем рабочем каталоге. Таким образом, вызов `os.Executable()` и `filepath.Dir()` позволяет определить путь реального исполняемого файла: если в пути встречается подстрока `go-build`, значит программа запущена через `go run`, иначе — это результат сборки через `go build`.  

Код для иллюстрации:  
```go
ex, _ := os.Executable()
dir := filepath.Dir(ex)
if strings.Contains(dir, "go-build") {
    fmt.Println("Запуск через go run")
} else {
    fmt.Println("Запуск через go build")
}
```  

Диаграмма:  
```mermaid
flowchart TD
A[os.Executable()] --> B{filepath.Dir}
B --> C{dir содержит "go-build"}
C -->|Да| D[go run]
C -->|Нет| E[go build]
```

```old
// GetWorkDir() - ex := os.Executable() >> dir := filepath.Dir(ex) >> strings.Contains(dir, "go-build") - как способ узнать: go build / go run
```