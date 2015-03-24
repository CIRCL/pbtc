package all

type nodeRepository struct {
	addrIn   chan string
	addrOut  chan<- string
	addrList map[string]bool
}

func NewNodeRepository() *nodeRepository {

	addrIn := make(chan string, bufferRepository)
	addrList := make(map[string]bool)

	nRepo := &nodeRepository{
		addrIn:   addrIn,
		addrList: addrList,
	}

	return nRepo
}

func (nRepo *nodeRepository) GetAddrIn() chan<- string {
	return nRepo.addrIn
}

func (nRepo *nodeRepository) Start(addrOut chan<- string) {

	nRepo.addrOut = addrOut

	go nRepo.handleAddresses()
}

func (nRepo *nodeRepository) Stop() {

	close(nRepo.addrIn)
}

func (nRepo *nodeRepository) handleAddresses() {

	for addr := range nRepo.addrIn {

		_, ok := nRepo.addrList[addr]
		if ok {
			continue
		}

		nRepo.addrList[addr] = true

		nRepo.addrOut <- addr
	}
}
