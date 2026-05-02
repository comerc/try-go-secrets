В Go конструкция `var _ sql.Scanner = (*Time)(nil)` используется как проверка на этапе компиляции: она заставляет компилятор убедиться, что тип `*Time` реализует интерфейс `sql.Scanner`. Если методы интерфейса реализованы неправильно или пропущены — программа не соберется. Фактически переменной `_` присваивается значение типа интерфейса, но это значение никогда не используется, поэтому код безопасен и не создает дополнительного рантайм-нагрузки.  

Таким образом обеспечивается контракт между вашим типом и интерфейсом. Это лучший способ заранее получать гарантию соответствия типов ожидаемому интерфейсу во всей кодовой базе. Подробности и примеры описаны в статье: https://medium.com/@matryer/golang-tip-compile-time-checks-to-ensure-your-type-satisfies-an-interface-c167afed3aae

```old
// var _ sql.Scanner = (*Time)(nil) - способ задать компилятору требование реализовать методы интерфейса sql.Scanner (Compile time checks to ensure your type satisfies an interface)[https://medium.com/@matryer/golang-tip-compile-time-checks-to-ensure-your-type-satisfies-an-interface-c167afed3aae]
```