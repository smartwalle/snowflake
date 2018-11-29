package xid

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	kSequenceBits   uint8 = 12 // 序列号占用的位数, 表示每个集群下的每个节点1毫秒内可生成的 id 序号的二进制位, 即每毫秒可生成 2^12-1=4096 个唯一 id
	kDataCenterBits uint8 = 5  // 数据中心占用的位数
	kWorkerBits     uint8 = 5  // 机器标识占用的位数

	kMaxSequence   int64 = -1 ^ (-1 << kSequenceBits)   // 序列号最大值，用于防止溢出
	kMaxDataCenter int64 = -1 ^ (-1 << kDataCenterBits) // 数据中心最大值，用于防止溢出
	kMaxWorker     int64 = -1 ^ (-1 << kWorkerBits)     // 机器标识最大值，用于防止溢出

	kTimeShift       = kDataCenterBits + kWorkerBits + kSequenceBits // 时间戳向左的偏移量
	kDataCenterShift = kWorkerBits + kSequenceBits                   // 数据中心向左的偏移量
	kMachineShift    = kSequenceBits                                 // 机器标识向左的偏移量
)

// --------------------------------------------------------------------------------
type Option interface {
	Apply(*SnowFlake) error
}

type optionFunc func(*SnowFlake) error

func (f optionFunc) Apply(s *SnowFlake) error {
	return f(s)
}

func WithDataCenter(dataCenter int64) Option {
	return optionFunc(func(s *SnowFlake) error {
		if dataCenter < 0 || dataCenter > kMaxDataCenter {
			return errors.New(fmt.Sprintf("data center can't be greater than %d or less than 0", kMaxDataCenter))
		}
		s.dataCenter = dataCenter
		return nil
	})
}

func WithMachine(machine int64) Option {
	return optionFunc(func(s *SnowFlake) error {
		if machine < 0 || machine > kMaxWorker {
			return errors.New(fmt.Sprintf("worker can't be greater than %d or less than 0", kMaxWorker))
		}
		s.machine = machine
		return nil
	})
}

func WithTimeOffset(offset time.Duration) Option {
	return optionFunc(func(s *SnowFlake) error {
		s.timeOffset = int64(offset)
		return nil
	})
}

// --------------------------------------------------------------------------------
type SnowFlake struct {
	mu           sync.Mutex
	milliseconds int64 // 上一次生成 id 的时间戳（毫秒）
	dataCenter   int64 // 数据中心 id
	machine      int64 // 机器标识 id
	sequence     int64 // 当前毫秒已经生成的 id 序列号 (从0开始累加) 1毫秒内最多生成 4096 个 id
	timeOffset   int64
}

func NewSnowFlake(opts ...Option) (*SnowFlake, error) {
	var sf = &SnowFlake{}
	sf.milliseconds = 0
	sf.sequence = 0
	sf.timeOffset = 0
	sf.dataCenter = 0
	sf.machine = 0

	var err error
	for _, opt := range opts {
		if err = opt.Apply(sf); err != nil {
			return nil, err
		}
	}
	return sf, nil
}

func (this *SnowFlake) Next() int64 {
	this.mu.Lock()
	defer this.mu.Unlock()

	var milliseconds = this.getMilliseconds()
	if milliseconds < this.milliseconds {
		return -1
	}

	if this.milliseconds == milliseconds {
		this.sequence = (this.sequence + 1) & kMaxSequence
		if this.sequence == 0 {
			milliseconds = this.getNextMilliseconds()
		}
	} else {
		this.sequence = 0
	}
	this.milliseconds = milliseconds
	var id = int64((milliseconds-this.timeOffset)<<kTimeShift | (this.dataCenter << kDataCenterShift) | (this.machine << kMachineShift) | (this.sequence))
	return id
}

func (this *SnowFlake) getNextMilliseconds() int64 {
	var mill = this.getMilliseconds()
	for mill < this.milliseconds {
		mill = this.getMilliseconds()
	}
	return mill
}

func (this *SnowFlake) getMilliseconds() int64 {
	return time.Now().UnixNano() / 1e6
}

// --------------------------------------------------------------------------------
var defaultSnowFlake, _ = NewSnowFlake()

func Next() int64 {
	return defaultSnowFlake.Next()
}
