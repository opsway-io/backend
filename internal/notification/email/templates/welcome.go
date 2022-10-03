package templates

type WelcomeTemplate struct {
	Name           string
	ActivationLink string
}

func (t *WelcomeTemplate) Subject() string {
	return "Welcome to Opsway"
}

func (t *WelcomeTemplate) HTML() string {
	return "Welcome " + t.Name + ", please activate your account by clicking on the following link: " + t.ActivationLink
}

func (t *WelcomeTemplate) PlainText() string {
	return "Welcome " + t.Name + ", please activate your account by clicking on the following link: " + t.ActivationLink
}
