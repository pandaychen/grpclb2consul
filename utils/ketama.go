package utils

//consistent hash之ketama算法实现

import (
	"hash/crc32"
	"hash/fnv"
	"sort"
	"strconv"
	"sync"
)

type HashFunc func(data []byte) uint32

const (
	DefaultReplicas = 16
)

type KetamaConsistent struct {
	sync.RWMutex
	hash       HashFunc
	replicas   int
	ringKeys   []int          //  Sorted keys(//最终RING)
	hashMap    map[int]string //最终ring上节点的映射
	serviceMap map[string][]string
}

func Fnvhvalue(server string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(server))
	return h.Sum32()
}

func NewKetama(srv_replicas int, fncb HashFunc) *KetamaConsistent {
	k := &KetamaConsistent{
		replicas:   srv_replicas, //service replicas
		hash:       fncb,
		hashMap:    make(map[int]string),
		serviceMap: make(map[string][]string),
		ringKeys:   make([]int, 0),
	}
	if k.replicas <= 0 {
		k.replicas = DefaultReplicas
	}
	if k.hash == nil {
		k.hash = crc32.ChecksumIEEE
	}
	return k
}

func (k *KetamaConsistent) IsEmpty() bool {
	k.Lock()
	defer k.Unlock()

	return len(k.ringKeys) == 0
}

//向CH中添加ServerNode（物理节点）
func (k *KetamaConsistent) AddSrvNode(srvnodes ...string) {
	k.Lock()
	defer k.Unlock()

	for _, node := range srvnodes {
		//扩容副本
		for i := 0; i < k.replicas; i++ {
			//将副本转变为ring上的key
			key := int(k.hash([]byte(strconv.Itoa(i) + node)))

			if _, ok := k.hashMap[key]; !ok {
				k.ringKeys = append(k.ringKeys, key)
			}
			k.hashMap[key] = node
			k.serviceMap[node] = append(k.serviceMap[node], strconv.Itoa(key))
		}
	}

	//方便二分查找，对ringKeys数组排序
	sort.Ints(k.ringKeys)
}

//有现网服务器宕机,需要将该server关联的所有key，从ring上移除
func (k *KetamaConsistent) RemoveSrvNode(nodes ...string) {
	k.Lock()
	defer k.Unlock()

	deletedkey_list := make([]int, 0)
	for _, node := range nodes {
		for i := 0; i < k.replicas; i++ {
			key := int(k.hash([]byte(strconv.Itoa(i) + node)))

			if _, ok := k.hashMap[key]; ok {
				deletedkey_list = append(deletedkey_list, key)
				delete(k.hashMap, key)
			}
		}
		//删除原有Srv节点的所有映射
		delete(k.serviceMap, node)
	}

	if len(deletedkey_list) > 0 {
		k.deleteKeys(deletedkey_list)
	}
}

//从ring(数组)中移除key，采用二分法较为高效
func (k *KetamaConsistent) deleteKeys(deletedKeysList []int) {

	//按升序排序
	sort.Ints(deletedKeysList)

	index := 0
	count := 0
	for _, key := range deletedKeysList {
		for ; index < len(k.ringKeys); index++ {
			k.ringKeys[index-count] = k.ringKeys[index]
			if key == k.ringKeys[index] {
				count++
				index++
				break
			}
		}
	}

	for ; index < len(k.ringKeys); index++ {
		k.ringKeys[index-count] = k.ringKeys[index]
	}

	k.ringKeys = k.ringKeys[:len(k.ringKeys)-count]
}

func (k *KetamaConsistent) GetSrvNode(client_key string) (string, bool) {
	if k.IsEmpty() {
		return "", false
	}
	k.RLock()
	defer k.RUnlock() //HERE must use  k.RUnlock() (core if use  k.Unlock() )

	//计算客户端传入的client_key的hash值
	hashval := int(k.hash([]byte(client_key)))

	//这里隐含了一层意思：k.keys[i] >= hash时，有可能计算出来的hash值比ring上的所有key值都大
	//为了找个一个Key应该放入哪个服务器，先哈希你的key，得到一个无符号整数, 沿着圆环找到和它相邻的最大的数，这个数对应的服务器就是被选择的服务器
	//对于靠近 2^32的 key, 因为没有超过它的数字点，按照圆环的原理，选择圆环中的第一个服务器
	//（通过这个函数，将一个可能不存在与ringkey数组的key（但是一定在环上），修正为离它最近的ringKey数组的key的下标）
	index := sort.Search(len(k.ringKeys), func(i int) bool {
		return k.ringKeys[i] >= hashval //if overflow,then returns len(k.ringKeys)
	})

	if index == len(k.ringKeys) {
		//it will core if not deal this case
		index = 0
	}

	conhash_key := k.ringKeys[index] //

	serveraddr, exsits := k.hashMap[conhash_key]
	return serveraddr, exsits
}
