package restarter

import (
	"context"
	"distribuidos/tp1/utils"
	"fmt"
	"slices"
)

func (r *Restarter) WaitLeader(amILeader bool) {
	r.condLeaderId.L.Lock()
	defer r.condLeaderId.L.Unlock()
	for amILeader != r.amILeader() {
		r.condLeaderId.Wait()
	}
}

// requires lock
func (r *Restarter) amILeader() bool {
	return (r.id == r.leaderId) && r.hasLeader
}

func (r *Restarter) startElection(ctx context.Context) error {
	log.Infof("Starting election")
	e := &Election{Ids: []uint64{r.id}}
	return r.sendToRing(ctx, e)
}

func (r *Restarter) handleElection(ctx context.Context, msg Election) error {
	log.Infof("Received Election message")

	if slices.Contains(msg.Ids, r.id) {
		return r.startCoordinator(ctx, msg.Ids)
	}
	msg.Ids = append(msg.Ids, r.id)

	return r.sendToRing(ctx, msg)
}

func (r *Restarter) startCoordinator(ctx context.Context, ids []uint64) error {
	log.Infof("Starting coordinator")
	leader := slices.Max(ids)
	r.condLeaderId.L.Lock()
	r.leaderId = leader
	r.hasLeader = true
	r.condLeaderId.L.Unlock()

	r.condLeaderId.Signal()

	log.Infof("Leader is %v", leader)

	coor := Coordinator{
		Leader: leader,
		Ids:    []uint64{r.id},
	}

	return r.sendToRing(ctx, coor)
}

func (r *Restarter) handleCoordinator(ctx context.Context, msg Coordinator) error {
	log.Infof("Received Coordinator message")

	if slices.Contains(msg.Ids, r.id) {
		return nil
	}

	r.condLeaderId.L.Lock()
	r.leaderId = msg.Leader
	r.hasLeader = true
	r.condLeaderId.L.Unlock()

	r.condLeaderId.Signal()

	log.Infof("Leader is %v", msg.Leader)

	msg.Ids = append(msg.Ids, r.id)
	return r.sendToRing(ctx, msg)
}

func (r *Restarter) sendToRing(ctx context.Context, msg Message) error {
	next := r.id + 1
	for {
		host := fmt.Sprintf("%v%v", RESTARTER_NAME, next%r.replicas)
		addr, _ := utils.GetUDPAddr(host, utils.RESTARTER_PORT)
		err := r.safeSend(ctx, msg, addr)
		if err == nil {
			return nil
		}
		log.Infof("Neighbor %v is not answering, sending message to next one", next%r.replicas)
		next += 1
	}
}
