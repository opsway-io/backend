package templates

type Template interface {
	Subject() string
	HTML() string
	PlainText() string
}
