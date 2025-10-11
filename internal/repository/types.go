package repository

// Entity — общий интерфейс, чтобы можно было работать с чем угодно
type Entity interface {
	TypeName() string
}