package controller

import (
	"fmt"
	"reflect"

	"github.com/Sirupsen/logrus"
	"github.com/openebs/jiva/replica/client"
	"github.com/openebs/jiva/types"
)

func getReplicaChain(address string) ([]string, error) {
	repClient, err := client.NewReplicaClient(address)
	if err != nil {
		return nil, fmt.Errorf("Cannot get replica client for %v: %v",
			address, err)
	}

	rep, err := repClient.GetReplica()
	if err != nil {
		return nil, fmt.Errorf("Cannot get replica for %v: %v",
			address, err)
	}
	return rep.Chain, nil
}

func (c *Controller) getCurrentAndRWReplica(address string) (*types.Replica, *types.Replica, error) {
	var (
		current, rwReplica *types.Replica
	)

	for i := range c.replicas {
		if c.replicas[i].Address == address {
			current = &c.replicas[i]
		} else if c.replicas[i].Mode == types.RW {
			rwReplica = &c.replicas[i]
		}
	}
	if current == nil {
		return nil, nil, fmt.Errorf("Cannot find replica %v", address)
	}
	if rwReplica == nil {
		return nil, nil, fmt.Errorf("Cannot find any healthy replica")
	}

	return current, rwReplica, nil
}

func (c *Controller) VerifyRebuildReplica(address string) error {
	// Prevent snapshot happens at the same time, as well as prevent
	// writing from happening since we're updating revision counter
	c.Lock()
	defer c.Unlock()

	replica, rwReplica, err := c.getCurrentAndRWReplica(address)
	if err != nil {
		return err
	}

	if replica.Mode == types.RW {
		return nil
	}
	if replica.Mode != types.WO {
		return fmt.Errorf("Invalid mode %v for replica %v to check", replica.Mode, address)
	}

	rwChain, err := getReplicaChain(rwReplica.Address)
	if err != nil {
		return err
	}

	logrus.Infof("chain %v from rw replica %s", rwChain, rwReplica.Address)
	// Don't need to compare the volume head disk
	rwChain = rwChain[1:]

	chain, err := getReplicaChain(address)
	if err != nil {
		return err
	}

	logrus.Infof("chain %v from wo replica %s", chain, address)
	chain = chain[1:]

	if !reflect.DeepEqual(rwChain, chain) {
		return fmt.Errorf("Replica %v's chain not equal to RW replica %v's chain",
			address, rwReplica.Address)
	}

	counter, err := c.backend.GetRevisionCounter(rwReplica.Address)
	if err != nil || counter == -1 {
		return fmt.Errorf("Failed to get revision counter of RW Replica %v: counter %v, err %v",
			rwReplica.Address, counter, err)

	}
	logrus.Infof("rw replica %s revision counter %d", rwReplica.Address, counter)

	if err := c.backend.SetRevisionCounter(address, counter); err != nil {
		return fmt.Errorf("Fail to set revision counter for %v: %v", address, err)
	}
	logrus.Debugf("WO replica %v's chain verified, update mode to RW", address)
	c.setReplicaModeNoLock(address, types.RW)
	if len(c.replicas) > c.replicaCount {
		logrus.Errorf("prev: replicacount %d len(replica) %d", c.replicaCount, len(c.replicas))
		c.replicaCount = len(c.replicas)
	}
	if len(c.quorumReplicas) > c.quorumReplicaCount {
		c.quorumReplicaCount = len(c.quorumReplicas)
	}
	c.UpdateVolStatus()
	return nil
}

func syncFile(from, to string, fromReplica, toReplica *types.Replica) error {
	if to == "" {
		to = from
	}

	fromClient, err := client.NewReplicaClient(fromReplica.Address)
	if err != nil {
		return fmt.Errorf("Cannot get replica client for %v: %v",
			fromReplica.Address, err)
	}

	toClient, err := client.NewReplicaClient(toReplica.Address)
	if err != nil {
		return fmt.Errorf("Cannot get replica client for %v: %v",
			toReplica.Address, err)
	}

	host, port, err := toClient.LaunchReceiver(to)
	if err != nil {
		return err
	}

	logrus.Infof("Synchronizing %s@%s to %s@%s:%d", from, fromReplica.Address, to, host, port)
	err = fromClient.SendFile(from, host, port)
	if err != nil {
		logrus.Infof("Failed synchronizing %s to %s@%s:%d: %v", from, to, host, port, err)
	} else {
		logrus.Infof("Done synchronizing %s to %s@%s:%d", from, to, host, port)
	}

	return err
}

func (c *Controller) PrepareRebuildReplica(address string) ([]string, error) {
	c.Lock()
	defer c.Unlock()

	if err := c.backend.SetRevisionCounter(address, 0); err != nil {
		return nil, fmt.Errorf("Fail to set revision counter for %v: %v", address, err)
	}

	replica, rwReplica, err := c.getCurrentAndRWReplica(address)
	if err != nil {
		return nil, err
	}
	if replica.Mode != types.WO {
		return nil, fmt.Errorf("Invalid mode %v for replica %v to prepare rebuild", replica.Mode, address)
	}

	rwChain, err := getReplicaChain(rwReplica.Address)
	if err != nil {
		return nil, err
	}

	fromHead := rwChain[0]

	chain, err := getReplicaChain(address)
	if err != nil {
		return nil, err
	}
	toHead := chain[0]

	if err := syncFile(fromHead+".meta", toHead+".meta", rwReplica, replica); err != nil {
		return nil, err
	}

	return rwChain[1:], nil
}
