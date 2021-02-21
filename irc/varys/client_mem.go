package varys

type memClient struct {
	varys *Varys
}

// NewMemClient returns an in-memory variant of varys
func NewMemClient() Client {
	return &memClient{varys: NewVarys()}
}

func (c *memClient) Setup(params SetupParams) error {
	return c.varys.Setup(params, nil)
}

func (c *memClient) GetUIDToNicks() (result map[string]string, err error) {
	err = c.varys.GetUIDToNicks(struct{}{}, &result)
	return
}

func (c *memClient) Connect(params ConnectParams) error {
	return c.varys.Connect(params, nil)
}

func (c *memClient) QuitIfConnected(uid string, quitMessage string) error {
	return c.varys.QuitIfConnected(QuitParams{uid, quitMessage}, nil)
}

func (c *memClient) SendRaw(uid string, params InterpolationParams, messages ...string) error {
	return c.varys.SendRaw(SendRawParams{uid, messages, params}, nil)
}

func (c *memClient) Nick(uid string, nick string) error {
	return c.varys.Nick(NickParams{uid, nick}, nil)
}
