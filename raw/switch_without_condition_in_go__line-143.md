В Go конструкция `switch` может использоваться без явного условия, что превращает её в более читабельную альтернативу цепочке `if-else if-else`. В таком случае каждая ветка `case` выступает как логическое выражение, и выполнение попадет в первую из них, которая даст `true`. Это особенно удобно, когда требуется проверить несколько вариантов условий и выбрать только один из них, сохраняя код чистым и удобным для чтения.  

Пример:  

```go
x := 15
switch {
case x < 0:
    fmt.Println("negative")
case x == 0:
    fmt.Println("zero")
case x > 0 && x < 10:
    fmt.Println("small positive")
default:
    fmt.Println("large positive")
}
```  

Диаграмма логики:  

```mermaid
flowchart TD
A[Начало] --> B{Условие x < 0}
B -- true --> C[negative]
B -- false --> D{Условие x == 0}
D -- true --> E[zero]
D -- false --> F{Условие x > 0 && x < 10}
F -- true --> G[small positive]
F -- false --> H[large positive]
```

```old
// switch без условия полезен тем, что может использоваться для проверки нескольких условий
```