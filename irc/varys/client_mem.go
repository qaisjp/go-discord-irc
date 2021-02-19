package varys

type memClient struct {
	varys *Varys
}

// NewMemClient returns an in-memory variant of varys
func NewMemClient() Client {
	return &memClient{}
}

func (c *memClient) AddPuppet(name string) (realname string, err error) {
	if err = c.varys.AddPuppet(name, &realname); err != nil {
		return
	}
	return
}
