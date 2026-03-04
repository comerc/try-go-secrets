`sync.Cond` в Go — это примитив синхронизации, позволяющий координировать работу горутин через ожидание определённых условий. Его метод `Broadcast()` используется для широковещательного уведомления: он будит сразу всех горутины, которые находятся в состоянии ожидания на условной переменной, а не только одну (как это делает `Signal()`). Обычно это применяется, когда произошло глобальное изменение состояния, и каждая из ожидающих горутин должна проверить условие заново.  

Пример упрощённой схемы работы:  

```mermaid
sequenceDiagram
    participant G1 as Goroutine 1
    participant G2 as Goroutine 2
    participant G3 as Goroutine 3
    participant Cond as sync.Cond

    G1->>Cond: Wait()
    G2->>Cond: Wait()
    G3->>Cond: Wait()
    Note over G1,G3: Все горутины ожидают условия
    Cond->>G1: Broadcast wake
    Cond->>G2: Broadcast wake
    Cond->>G3: Broadcast wake
    Note over G1,G3: Все проснувшиеся горутины<br>проверяют условие
```

```old
// sync.Cond тоже умеет в широковещательное оповещение всех слушателей - .Broadcast()
```