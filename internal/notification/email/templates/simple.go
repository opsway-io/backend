package templates

type SimpleTemplate struct {
	SubjectText string
	Body        string
}

func (t *SimpleTemplate) Subject() string {
	return t.SubjectText
}

func (t *SimpleTemplate) HTML() string {
	return t.Body
}

func (t *SimpleTemplate) PlainText() string {
	return t.Body
}
