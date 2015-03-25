package all

import (
	"net"
)

type repository struct {
	addrIn   chan string
	connIn   chan net.Conn
	nodeList map[string]*node
}

func NewRepository() *repository {
	addrIn := make(chan string)
	connIn := make(chan net.Conn)
	nodeList := make(map[string]*node)

	repo := &repository{
		addrIn:   addrIn,
		connIn:   connIn,
		nodeList: nodeList,
	}

	return repo
}

func (repo *repository) FindByAddress(nodeOut chan<- *node, addr string, limit int) {
	numFound := 0

	for nodeAddr, node := range repo.nodeList {
		if numFound >= limit {
			break
		}

		if nodeAddr == addr {
			numFound++
			nodeOut <- node
		}
	}
}

func (repo *repository) FindByIP(nodeOut chan<- *node, ip string, limit int) {
	numFound := 0

	for _, node := range repo.nodeList {
		if numFound >= limit {
			break
		}

		nodeIp, _, err := net.SplitHostPort(node.Addr)
		if err != nil {
			continue
		}

		if nodeIp == ip {
			numFound++
			nodeOut <- node
		}
	}
}

func (repo *repository) FindByPort(nodeOut chan<- *node, port string, limit int) {
	numFound := 0

	for _, node := range repo.nodeList {
		if numFound >= limit {
			break
		}

		_, nodePort, err := net.SplitHostPort(node.Addr)
		if err != nil {
			continue
		}

		if nodePort == port {
			numFound++
			nodeOut <- node
		}
	}
}

func (repo *repository) FindByState(nodeOut chan<- *node, state uint32, limit int) {
	numFound := 0

	for _, node := range repo.nodeList {
		if numFound >= limit {
			break
		}

		if node.GetState() == state {
			numFound++
			nodeOut <- node
		}
	}
}

func (repo *repository) Start() {
	go repo.handleAddresses()
	go repo.handleConnections()
}

func (repo *repository) Stop() {
	close(repo.addrIn)
	close(repo.connIn)
}

func (repo *repository) GetAddrIn() chan<- string {
	return repo.addrIn
}

func (repo *repository) GetConnIn() chan<- net.Conn {
	return repo.connIn
}

func (repo *repository) handleAddresses() {
	for addr := range repo.addrIn {
		_, ok := repo.nodeList[addr]
		if ok {
			continue
		}

		node := NewNode(addr)
		repo.nodeList[addr] = node
	}
}

func (repo *repository) handleConnections() {
	for conn := range repo.connIn {
		addr := conn.RemoteAddr().String()
		_, ok := repo.nodeList[addr]
		if ok {
			continue
		}

		node := NewNode(addr)
		node.UseConnection(conn)
		repo.nodeList[addr] = node
	}
}
