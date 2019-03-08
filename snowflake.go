package snowflake

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	kSequenceBits   uint8 = 12 // 序列号占用的位数, 表示每个集群下的每个节点1毫秒内可生成的 id 序号的二进制位, 即每毫秒可生成 2^12-1=4096 个唯一 id
	kDataCenterBits uint8 = 5  // 数据中心占用的位数
	kMachineBits    uint8 = 5  // 机器标识占用的位数

	kMaxSequence   int64 = -1 ^ (-1 << kSequenceBits)   // 序列号最大值，用于防止溢出   0-4095
	kMaxDataCenter int64 = -1 ^ (-1 << kDataCenterBits) // 数据中心最大值，用于防止溢出 0-31
	kMaxMachine    int64 = -1 ^ (-1 << kMachineBits)    // 机器标识最大值，用于防止溢出 0-31

	kTimeShift       = kDataCenterBits + kMachineBits + kSequenceBits // 时间戳向左的偏移量
	kDataCenterShift = kMachineBits + kSequenceBits                   // 数据中心向左的偏移量
	kMachineShift    = kSequenceBits                                  // 机器标识向左的偏移量

	kDataCenterMask = kMaxDataCenter << kDataCenterShift
	kMachineMask    = kMaxMachine << kSequenceBits
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
		if machine < 0 || machine > kMaxMachine {
			return errors.New(fmt.Sprintf("worker can't be greater than %d or less than 0", kMaxMachine))
		}
		s.machine = machine
		return nil
	})
}

// WithTimeOffset offset 值为 millisecond
func WithTimeOffset(t time.Time) Option {
	return optionFunc(func(s *SnowFlake) error {
		if t.IsZero() {
			return nil
		}
		s.timeOffset = t.UnixNano() / 1e6
		return nil
	})
}

// --------------------------------------------------------------------------------
type SnowFlake struct {
	mu          sync.Mutex
	millisecond int64 // 上一次生成 id 的时间戳（毫秒）
	dataCenter  int64 // 数据中心 id
	machine     int64 // 机器标识 id
	sequence    int64 // 当前毫秒已经生成的 id 序列号 (从0开始累加) 1毫秒内最多生成 4096 个 id
	timeOffset  int64
}

func New(opts ...Option) (*SnowFlake, error) {
	var sf = &SnowFlake{}
	sf.millisecond = 0
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

	var millisecond = this.getMillisecond()
	if millisecond < this.millisecond {
		return -1
	}

	if this.millisecond == millisecond {
		this.sequence = (this.sequence + 1) & kMaxSequence
		if this.sequence == 0 {
			millisecond = this.getNextMillisecond()
		}
	} else {
		this.sequence = 0
	}
	this.millisecond = millisecond

	var id = int64((millisecond-this.timeOffset)<<kTimeShift | (this.dataCenter << kDataCenterShift) | (this.machine << kMachineShift) | (this.sequence))
	return id
}

func (this *SnowFlake) getNextMillisecond() int64 {
	var mill = this.getMillisecond()
	for mill < this.millisecond {
		mill = this.getMillisecond()
	}
	return mill
}

func (this *SnowFlake) getMillisecond() int64 {
	return time.Now().UnixNano() / 1e6
}

// --------------------------------------------------------------------------------
// Time 获取 id 的时间差值，单位是 millisecond
func Time(s int64) int64 {
	return s >> kTimeShift
}

// DataCenter 获取 id 的数据中心标识
func DataCenter(s int64) int64 {
	return s & kDataCenterMask >> kDataCenterShift
}

// Machine 获取 id 的机器标识
func Machine(s int64) int64 {
	return s & kMachineMask >> kMachineShift
}

//  Sequence 获取 id 的序列号
func Sequence(s int64) int64 {
	return s & kMaxSequence
}

// --------------------------------------------------------------------------------
var defaultSnowFlake *SnowFlake
var once sync.Once

func Next() int64 {
	once.Do(func() {
		defaultSnowFlake, _ = New()
	})
	return defaultSnowFlake.Next()
}

func Init(opts ...Option) (err error) {
	once.Do(func() {
		defaultSnowFlake, err = New(opts...)
	})

	if err != nil {
		once = sync.Once{}
	}

	return err
}
