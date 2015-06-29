// Copyright (c) 2015 Max Wolter
// Copyright (c) 2015 CIRCL - Computer Incident Response Center Luxembourg
//                           (c/o smile, security made in Lëtzebuerg, Groupement
//                           d'Intérêt Economique)
//
// This file is part of PBTC.
//
// PBTC is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// PBTC is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with PBTC.  If not, see <http://www.gnu.org/licenses/>.

package adaptor

// Manager defines the interface used by peers to communicate with their
// manager. It is notified of peer state, keeps track of shared state and
// decides on actions depending on state. Different managers can implement
// different behaviours.
type Manager interface {
	Start()
	Stop()
	SetLog(Log)
	SetRepository(Repository)
	SetTracker(Tracker)
	AddProcessor(Processor)
	Connected(Peer)
	Ready(Peer)
	Stopped(Peer)
}
