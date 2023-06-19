package calico

func (h *Handler) CreateNetworkPolicy(ns string) (err error) {
	_, err = h.cali.Policy().Create(ns)
	return
}

func (h *Handler) DeleteNetworkPolicy(ns string) (err error) {
	return h.cali.Policy().Delete(ns)
}
