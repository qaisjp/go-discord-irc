package transmitter

import (
	"container/heap"
)

type webhookHeap struct {
	indices map[string]int
	list    []webhook
}

func newWebhookHeap() webhookHeap {
	return webhookHeap{
		make(map[string]int),
		[]webhook{},
	}
}

func (h webhookHeap) Len() int {
	return len(h.list)
}

func (h webhookHeap) Less(i, j int) bool {
	return h.list[i].lastUse.Before(h.list[j].lastUse)
}

func (h webhookHeap) Swap(i, j int) {
	iChannel := h.list[i].ChannelID
	jChannel := h.list[j].ChannelID

	h.list[i], h.list[j] = h.list[j], h.list[i]
	h.indices[iChannel], h.indices[jChannel] = j, i
}

func (h *webhookHeap) Push(x interface{}) {
	wh := x.(*wrappedWebhook)
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	h.indices[wh.ChannelID] = len(h.list)
	h.list = append(h.list, wh)
}

func (h *webhookHeap) Pop() interface{} {
	old := h.list
	n := len(old)
	x := old[n-1]
	h.list = old[0 : n-1]
	delete(h.indices, x.ChannelID)
	return x
}

func (h *webhookHeap) Remove(channel string) {
	i := h.indices[channel]
	heap.Remove(h, i)
}

func (h *webhookHeap) Get(channel string) webhook {
	i, ok := h.indices[channel]
	if !ok {
		return nil
	}

	return h.list[i]
}

// Fix must to be called when a webhook's
// lastUse attribute is changed.
//
// This function will panic if no such
// webhook for that channel exists.
func (h *webhookHeap) Fix(channel string) {
	i, ok := h.indices[channel]
	if !ok {
		panic("could not fix channel: " + channel)
	}

	heap.Fix(h, i)
}
