package pkg

import (
	"fmt"
	"strconv"

	"github.com/RoaringBitmap/roaring"
)

func Cache(storage Storage) error {
	uncachedNodes, err := storage.ToBeCached()
	if err != nil {
		return err
	}
	if len(uncachedNodes) == 0 {
		return nil
	}
	keys, err := storage.GetAllKeys()
	if err != nil {
		return fmt.Errorf("error getting keys: %w", err)
	}

	// Retrieve all nodes at once
	allNodes, err := storage.GetNodes(keys)
	if err != nil {
		return fmt.Errorf("error getting all nodes: %w", err)
	}

	childSCC, err := findCycles(storage, ChildrenDirection, len(keys), allNodes)
	if err != nil {
		return err
	}

	cachedChildren, err := buildCache(storage, uncachedNodes, ChildrenDirection, childSCC, allNodes)
	if err != nil {
		return err
	}

	parentSCC, err := findCycles(storage, ParentsDirection, len(keys), allNodes)
	if err != nil {
		return err
	}

	cachedParents, err := buildCache(storage, uncachedNodes, ParentsDirection, parentSCC, allNodes)
	if err != nil {
		return err
	}

	cachedChildKeys, cachedChildValues, err := cachedChildren.GetAllKeysAndValues()
	if err != nil {
		return err
	}

	var caches []*NodeCache

	for i := 0; i < len(cachedChildKeys); i++ {
		childId := cachedChildKeys[i]
		childIntId, err := strconv.Atoi(childId)
		if err != nil {
			return err
		}

		childBindValue := cachedChildValues[i].Clone()
		childBindValue.Remove(uint32(childIntId))

		tempValue, err := cachedParents.Get(strconv.Itoa(childIntId))
		if err != nil {
			return fmt.Errorf("error getting value for key %s, err: %v", childId, err)
		}
		parentBindValue := tempValue.Clone()
		parentBindValue.Remove(uint32(childIntId))
		caches = append(caches, NewNodeCache(uint32(childIntId), parentBindValue, childBindValue))
	}

	if err := storage.SaveCaches(caches); err != nil {
		return err
	}
	return storage.ClearCacheStack()
}

func findCycles(storage Storage, direction Direction, numOfNodes int, allNodes map[uint32]*Node) (map[uint32]uint32, error) {
	var stack []uint32
	var tarjanDFS func(nodeID uint32) error

	currentTarjanID := 0
	nodeToTarjanID := map[uint32]uint32{}
	lowLink := make(map[uint32]uint32)
	inStack := roaring.New()

	tarjanDFS = func(nodeID uint32) error {
		currentNode := allNodes[nodeID]

		var nextNodes []uint32
		if direction == ChildrenDirection {
			nextNodes = currentNode.Children.ToArray()
		} else {
			nextNodes = currentNode.Parents.ToArray()
		}

		currentTarjanID++
		stack = append(stack, nodeID)
		inStack.Add(nodeID)
		nodeToTarjanID[nodeID] = uint32(currentTarjanID)
		lowLink[nodeID] = uint32(currentTarjanID)

		for _, nextNode := range nextNodes {
			if _, visited := nodeToTarjanID[nextNode]; !visited {
				if err := tarjanDFS(nextNode); err != nil {
					return err
				}
				lowLink[nodeID] = min(lowLink[nodeID], lowLink[nextNode])
			} else if inStack.Contains(nextNode) {
				lowLink[nodeID] = min(lowLink[nodeID], nodeToTarjanID[nextNode])
			}
		}

		if nodeToTarjanID[nodeID] == lowLink[nodeID] {
			for len(stack) > 0 {
				id := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				inStack.Remove(id)
				lowLink[id] = nodeToTarjanID[nodeID]
				if nodeID == id {
					break
				}
			}
		}
		return nil
	}

	for id := 1; id < numOfNodes+1; id++ {
		if _, visited := nodeToTarjanID[uint32(id)]; !visited {
			if err := tarjanDFS(uint32(id)); err != nil {
				return nil, err
			}
		}
	}

	return lowLink, nil
}

func buildCache(storage Storage, uncachedNodes []uint32, direction Direction, scc map[uint32]uint32, allNodes map[uint32]*Node) (*NativeKeyManagement, error) {
	cache, children, parents := NewNativeKeyManagement(), NewNativeKeyManagement(), NewNativeKeyManagement()
	alreadyCached := roaring.New()

	err := addCyclesToBindMap(storage, scc, cache, children, parents, allNodes)
	if err != nil {
		return nil, err
	}

	for _, nodeID := range uncachedNodes {
		if err := cacheDFS(storage, nodeID, direction, scc, alreadyCached, cache, children, parents, allNodes); err != nil {
			return nil, err
		}
	}
	return cache, nil
}

func cacheDFS(storage Storage, nodeID uint32, direction Direction, scc map[uint32]uint32, alreadyCached *roaring.Bitmap, cache, children, parents *NativeKeyManagement, allNodes map[uint32]*Node) error {
	curNode := allNodes[nodeID]

	if alreadyCached.Contains(curNode.ID) {
		return nil
	}

	todoNodes, futureNodes, err := getTodoAndFutureNodes(children, parents, curNode, direction)
	if err != nil {
		return err
	}

	for _, id := range todoNodes {
		if !(scc[curNode.ID] == scc[id]) {
			if alreadyCached.Contains(id) {
				if err := addToCache(cache, nodeID, id); err != nil {
					return err
				}
			} else if err := cacheDFS(storage, id, direction, scc, alreadyCached, cache, children, parents, allNodes); err != nil {
				return err
			}
		}
	}

	alreadyCached.Add(nodeID)

	// We have to iterate through the future nodes
	for _, id := range futureNodes {
		if err := cacheDFS(storage, id, direction, scc, alreadyCached, cache, children, parents, allNodes); err != nil {
			return err
		}
	}

	return nil
}

func addCyclesToBindMap(storage Storage, scc map[uint32]uint32, cache, children, parents *NativeKeyManagement, allNodes map[uint32]*Node) error {
	parentToKeys := map[uint32][]string{}

	for k, v := range scc {
		parentToKeys[v] = append(parentToKeys[v], strconv.Itoa(int(k)))
	}

	for _, keysForAParent := range parentToKeys {
		if _, err := cache.BindKeys(keysForAParent); err != nil {
			return err
		}
		if _, err := children.BindKeys(keysForAParent); err != nil {
			return err
		}
		if _, err := parents.BindKeys(keysForAParent); err != nil {
			return err
		}

		keycache := roaring.New()
		childrenCache, parentCache := roaring.New(), roaring.New()
		for _, key := range keysForAParent {
			intkey, err := strconv.Atoi(key)
			if err != nil {
				return err
			}

			node := allNodes[uint32(intkey)]

			childrenCache.Or(node.Children)
			parentCache.Or(node.Parents)
			keycache.Add(uint32(intkey))
			keycache.Add(node.ID)
		}
		childrenCache.AndNot(keycache)
		parentCache.AndNot(keycache)
		if len(keysForAParent) > 0 {
			if err := cache.Set(keysForAParent[0], *keycache); err != nil {
				return fmt.Errorf("error setting value for key in cache %s, %w", keysForAParent[0], err)
			}
			if err := children.Set(keysForAParent[0], *childrenCache); err != nil {
				return fmt.Errorf("error setting value for key in grouped children relationship%s, %w", keysForAParent[0], err)
			}
			if err := parents.Set(keysForAParent[0], *parentCache); err != nil {
				return fmt.Errorf("error setting value for key in grouped parent relationship%s, %w", keysForAParent[0], err)
			}
		}

	}
	return nil
}

func getTodoAndFutureNodes(children, parents *NativeKeyManagement, curNode *Node, direction Direction) ([]uint32, []uint32, error) {
	var todoNodes, futureNodes []uint32

	if direction == ChildrenDirection {
		todoNodesBitmap, err := children.Get(strconv.Itoa(int(curNode.ID)))
		if err != nil {
			return nil, nil, err
		}
		futureNodesBitmap, err := parents.Get(strconv.Itoa(int(curNode.ID)))
		if err != nil {
			return nil, nil, err
		}
		todoNodes, futureNodes = todoNodesBitmap.ToArray(), futureNodesBitmap.ToArray()
	} else {
		todoNodesBitmap, err := parents.Get(strconv.Itoa(int(curNode.ID)))
		if err != nil {
			return nil, nil, err
		}
		futureNodesBitmap, err := children.Get(strconv.Itoa(int(curNode.ID)))
		if err != nil {
			return nil, nil, err
		}
		todoNodes, futureNodes = todoNodesBitmap.ToArray(), futureNodesBitmap.ToArray()
	}

	return todoNodes, futureNodes, nil
}

func addToCache(bm *NativeKeyManagement, curElem, todoElem uint32) error {
	curElemVal, err := bm.Get(strconv.Itoa(int(curElem)))
	if err != nil {
		return fmt.Errorf("error getting value for curElem key from value %d, err: %v", curElem, err)
	}

	todoVal, err := bm.Get(strconv.Itoa(int(todoElem)))
	if err != nil {
		return fmt.Errorf("error getting value for curElem key %d, err: %v", todoElem, err)
	}

	curElemVal.Or(&todoVal)
	curElemVal.Add(curElem)

	err = bm.Set(strconv.Itoa(int(curElem)), curElemVal)
	if err != nil {
		return err
	}
	return nil
}
