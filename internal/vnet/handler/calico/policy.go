package calico

func (h *Handler) CreateNetworkPolicy(labels map[string]string) (err error) {
	ns, ok := labels["namespace"]
	if ok {
		_, err = h.cali.Policy().Create(ns)
		if err != nil {
			return
		}
	}
	return
}

func (h *Handler) DeleteNetworkPolicy(labels map[string]string) (err error) {
	ns, ok := labels["namespace"]
	if ok {
		err = h.cali.Policy().Delete(ns)
		if err != nil {
			return
		}
	}
	return
}
