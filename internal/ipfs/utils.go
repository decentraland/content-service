package ipfs

import (
	"github.com/ipsn/go-ipfs/core"
	"github.com/ipsn/go-ipfs/core/coreunix"
	"io"
)

type IpfsHelper struct {
	Node *core.IpfsNode
}

func (h *IpfsHelper) CalculateCID(reader io.Reader) (string, error) {
	actualCID, err := coreunix.Add(h.Node, reader)
	if err != nil {
		return "", err
	}
	return actualCID, nil
}
