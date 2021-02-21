package varys

type memClient struct {
	varys *Varys
}

// NewMemClient returns an in-memory variant of varys
func NewMemClient() Client {
	return &memClient{varys: &Varys{}}
}

func (c *memClient) Setup(params SetupParams) error {
	return c.varys.Setup(params, nil)
}

func (c *memClient) GetUIDToNicks() (result map[string]string, err error) {
	err = c.varys.GetUIDToNicks(struct{}{}, &result)
	return
}
