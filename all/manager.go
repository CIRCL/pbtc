package all

import (
	"log"

	"github.com/btcsuite/btcd/wire"
)

type dataManager struct {
	peerIn   chan *peer
	addrOut  chan<- string
	peerList map[*peer]struct{}
}

func NewDataManager() *dataManager {

	peerIn := make(chan *peer, bufferManager)
	peerList := make(map[*peer]struct{})

	dManager := &dataManager{
		peerIn:   peerIn,
		peerList: peerList,
	}

	return dManager
}

func (dManager *dataManager) GetPeerIn() chan<- *peer {
	return dManager.peerIn
}

func (dManager *dataManager) Start(addrOut chan<- string) {

	dManager.addrOut = addrOut

	go dManager.handlePeers()
}

func (dManager *dataManager) Stop() {

	close(dManager.peerIn)
}

func (dManager *dataManager) handlePeers() {

	for peer := range dManager.peerIn {

		_, ok := dManager.peerList[peer]
		if ok {
			log.Println("Peer already known")
			continue
		}

		dManager.peerList[peer] = struct{}{}
		log.Println("New peer added, total:", len(dManager.peerList))

		msg := wire.NewMsgGetAddr()
		peer.SendMessage(msg)

		peer.Process(dManager.addrOut)
	}
}
