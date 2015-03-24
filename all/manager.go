package all

import (
	"log"
)

type dataManager struct {
	peerIn chan *peer
}

func NewDataManager() *dataManager {

	peerIn := make(chan *peer, bufferManager)

	dManager := &dataManager{
		peerIn: peerIn,
	}

	return dManager
}

func (dManager *dataManager) GetPeerIn() chan<- *peer {
	return dManager.peerIn
}

func (dManager *dataManager) Start() {

	go dManager.handlePeers()
}

func (dManager *dataManager) Stop() {

	close(dManager.peerIn)
}

func (dManager *dataManager) handlePeers() {

	for peer := range dManager.peerIn {

		log.Println("New peer connected", peer)
	}
}
