package util

import "github.com/projecteru2/libyavirt/types"

func VirtID(id string) string {
	req := types.GuestReq{ID: id}
	return req.VirtID()
}
