package main

import (
	"encoding/hex"
	"log"
	"reflect"
	"syscall"
	"unsafe"
)

const MbSize = 1024 * 1024
const PathMax = 4096

func ceil(n float32) int {
	return int(n) + 1
}

//super_block的信息
type superBlock struct {
	sMagic          uint32 //4字节0xabcdefa
	sBlockSize      uint32 //每个块大小
	sBlockCounts    uint32 //块的总数量
	sDataBlockFirst uint32 //第一个数据块的偏移量
	sDevSize        uint32 //磁盘的大小
	sOffset         uint32 //信息块的偏移量
	sMkfsTime       uint32 //文件系统被创建的时间
	reserve         [512 - 28]byte
}

//数据块的信息
type blockInfo struct {
	bCreatTime  uint32 //数据块被创建的时间
	bUpdateTime uint32 //数据块更新的时间
	bCsmLength  uint32 //写入文件的大小
	bDataLength uint32
	bBlockId    uint32 //bBlockId
}

type blockExtraInfo struct {
	path [PathMax]byte
}

// BlockInfoSize 信息块部分大小
func BlockInfoSize() uintptr {
	return 1 + unsafe.Sizeof(blockInfo{}) + unsafe.Sizeof(blockExtraInfo{})
}

// BlockNum 数据块的个数
func BlockNum(devsize uintptr, blocksize uintptr) uintptr {
	return (devsize - 512) / (blocksize + BlockInfoSize()) //数据块的个数
}

// BitmapSize bitmap 大小
func BitmapSize(devsize uintptr, blocksize uintptr) uintptr {
	return uintptr(ceil(float32(BlockNum(devsize, blocksize) / 8.0))) //bitmap 大小
}

// BitmapSpace bitmap 占用的512块数
func BitmapSpace(devsize uintptr, blocksize uintptr) uintptr {
	return uintptr(ceil(float32(BitmapSize(devsize, blocksize) / 512.0))) //bitmap 占用的512块数
}

// BitmapOffset bitmap 偏移量
func BitmapOffset() uintptr {
	return unsafe.Sizeof(superBlock{})
}

// InfoSize block info 大小
func InfoSize(devsize uintptr, blocksize uintptr) uintptr {
	return BlockNum(devsize, blocksize) * unsafe.Sizeof(blockInfo{}) //block info 大小
}

// InfoSpace block info 占用的512块数
func InfoSpace(devsize uintptr, blocksize uintptr) uintptr {
	return uintptr(ceil(float32(InfoSize(devsize, blocksize) / 512.0))) //block info 占用的512块数
}

// InfoOffset block info 起始偏移量
func InfoOffset(devsize uintptr, blocksize uintptr) uintptr {
	return BitmapOffset() + BitmapSpace(devsize, blocksize)*512 //block info 起始偏移量)
}

// ExtraSize block extra info 大小
func ExtraSize(devsize uintptr, blocksize uintptr) uintptr {
	return BlockNum(devsize, blocksize) * unsafe.Sizeof(blockExtraInfo{}) //block extra info 大小
}

// ExtraSpace block extra info 占用的512块数
func ExtraSpace(devsize uintptr, blocksize uintptr) uintptr {
	return uintptr(ceil(float32(ExtraSize(devsize, blocksize) / 512.0))) //block extra info 占用的512块数
}

// ExtraOffset block extra inf 起始偏移量
func ExtraOffset(devsize uintptr, blocksize uintptr) uintptr {
	return InfoOffset(devsize, blocksize) + InfoSpace(devsize, blocksize)*512 //block extra inf 起始偏移量
}

// DataOffset 数据块的起始偏移量
func DataOffset(devsize uintptr, blocksize uintptr) uintptr {
	return ExtraOffset(devsize, blocksize) + ExtraSpace(devsize, blocksize)*512 //数据块的起始偏移量
}

var sizeOfsuperBlock = int(unsafe.Sizeof(superBlock{}))

func superBlockToBytes(sb *superBlock) []byte {
	var x reflect.SliceHeader
	x.Len = sizeOfsuperBlock
	x.Cap = sizeOfsuperBlock
	x.Data = uintptr(unsafe.Pointer(sb))
	return *(*[]byte)(unsafe.Pointer(&x))
}
func blockInfoToBytes(bii *blockInfo) []byte {
	var x reflect.SliceHeader
	x.Len = int(unsafe.Sizeof(blockInfo{}))
	x.Cap = int(unsafe.Sizeof(blockInfo{}))
	x.Data = uintptr(unsafe.Pointer(bii))
	return *(*[]byte)(unsafe.Pointer(&x))
}

func BytesToSuperBlock(b []byte) *superBlock {
	return (*superBlock)(unsafe.Pointer(
		(*reflect.SliceHeader)(unsafe.Pointer(&b)).Data,
	))
}
func BytesToBlockInfo(b []byte) *blockInfo {
	return (*blockInfo)(unsafe.Pointer(
		(*reflect.SliceHeader)(unsafe.Pointer(&b)).Data,
	))
}

// ReadSuperBlock 读取superblock
func ReadSuperBlock(fd int) (sb *superBlock) {

	var bytetest = make([]byte, unsafe.Sizeof(superBlock{}))
	_, _ = syscall.Read(fd, bytetest)

	//log.Println("/dev/sdb test", bytetest)
	//log.Println("num:", num)

	//[]byte转16进制输出
	//encodedtest := hex.EncodeToString(bytetest)
	//log.Println(encodedtest)

	sb = BytesToSuperBlock(bytetest)
	//fmt.Println(sb)
	//log.Println("block_size", sb.sBlockSize)
	//log.Println("sb.sMagic", sb.sMagic)
	//log.Println("sb.sBlockCounts", sb.sBlockCounts)
	//log.Println("dev_size", sb.sDevSize)

	return sb
}

//读取bitmap
func readBitmap(fd int, devSize uint32, blockSize uint32) (pBitmap []byte) {
	len := BitmapSize(uintptr(devSize), uintptr(blockSize))
	pBitmap = make([]byte, len)
	//log.Println("len:", len)
	offset := BitmapOffset()
	//log.Println("offset", offset)
	syscall.Pread(fd, pBitmap, int64(offset))
	//log.Println("p_bitmap:", pBitmap)
	encodeBitmap := hex.EncodeToString(pBitmap)
	log.Println("BitMap:", encodeBitmap)
	log.Println("bitmap的len:", len)
	return pBitmap
}

//读取blockinfo
func readblockInfo(fd int, devSize uint32, blockSize uint32) (bis []blockInfo) {
	//bit_size := BLOCK_NUM(uintptr(dev_size), uintptr(block_size))
	//判断path传入的是csm还是data来确定写入那个数据
	//待实现

	//读取block_info,运用读取数据的原理，找到data_block_id
	//根据bitmap找到有效的b_block_id列表
	biLen := InfoSize(uintptr(devSize), uintptr(blockSize))
	offset := InfoOffset(uintptr(devSize), uintptr(blockSize))
	var bytetest = make([]byte, biLen)
	_, err := syscall.Pread(fd, bytetest, int64(offset))
	if err != nil {
		log.Fatal("读取blockInfo失败", err)
	}
	var bi *blockInfo
	for i := 0; i < len(bytetest); {
		bi = BytesToBlockInfo(bytetest[i : i+int(unsafe.Sizeof(blockInfo{}))])
		bis = append(bis, *bi)
		i = i + int(unsafe.Sizeof(blockInfo{}))
	}

	//bi = BytesToBlockInfo(bytetest)
	return bis
}

//获取有效b_block_id列表
func bitmapUsedblockid(binaryBytes []byte) (usedBlocklist []int) {
	for i, v := range binaryBytes {
		log.Println("i,v:", i, v)
		if v == 49 {
			usedBlocklist = append(usedBlocklist, i)
		}
	}
	return usedBlocklist

}
func bitmapUsedBlockId(bitmap []byte, bitSize uintptr) (usedBlocklist []int) {
	for i := 0; i < int(bitSize); i++ {
		if bitmapGet(i, bitmap, bitSize) > 0 {
			usedBlocklist = append(usedBlocklist, i)
		}
	}

	return usedBlocklist
}
func bitmapAvailableBlockId(bitmap []byte, bitSize uintptr) (availableList []int) {
	for i := 0; i < int(bitSize); i++ {
		if bitmapGet(i, bitmap, bitSize) == 0 {
			availableList = append(availableList, i)
		}
	}
	return availableList
}
func bitmapAvailableblockid(binaryBytes []byte) (availableList []int) {
	for i, binaryByte := range binaryBytes {
		log.Println("i,binaryByte", i, binaryByte)
		if binaryByte == 48 {
			availableList = append(availableList, i)
		}
	}
	return availableList
}

//返回第index位对应的值
func bitmapGet(index int, bitmap []byte, bitSize uintptr) (ret uint32) {
	if index >= int(bitSize) {
		return 0
	}
	i := index / 8
	v := bitmap[i]
	r := index % 8
	var mask uint8 = 0x01
	mask <<= r
	if v&mask != 0 {
		return 1
	} else {
		return 0
	}
}
func bitmapSet(index int, oldBitmap []byte, fd int, bitSize uintptr) {
	if index >= int(bitSize) {
		log.Println("index越位")
	}
	offset := BitmapOffset()
	i := index / 8
	v := oldBitmap[i]
	r := index % 8
	var mask uint8 = 0x01
	mask <<= r
	v = v | mask
	oldBitmap[i] = v
	//log.Println(oldBitmap)
	_, err := syscall.Pwrite(fd, oldBitmap, int64(offset))
	if err != nil {
		log.Fatal("bitmap更新失败-添加")
	}
}
func bitmapUnset(index int, oldBitmap []byte, fd int, bitSize uintptr) {
	if index >= int(bitSize) {
		log.Println("index越位")
	}
	offset := BitmapOffset()
	i := index / 8
	v := oldBitmap[i]
	r := index % 8
	var mask uint8 = 0x01
	mask <<= r
	v = v ^ mask
	oldBitmap[i] = v
	//log.Println(oldBitmap)
	_, err := syscall.Pwrite(fd, oldBitmap, int64(offset))
	if err != nil {
		log.Fatal("bitmap更新失败-删除")
	}
}
func bitmapSetTest(index int, oldBitmap []byte) []byte {
	i := index / 8
	v := oldBitmap[i]
	r := index % 8
	var mask uint8 = 0x01
	mask <<= r
	v |= mask
	oldBitmap[i] = v
	return oldBitmap
}
func bitmapUnsetetTest(index int, oldBitmap []byte) []byte {
	i := index / 8
	v := oldBitmap[i]
	r := index % 8
	var mask uint8 = 0x01
	mask <<= r
	v ^= mask
	oldBitmap[i] = v
	return oldBitmap
}

//
//func main() {
//	fd, err := syscall.Open("/dev/sdb", os.O_RDWR, 0777)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	//测试系统启动
//	//files:=startFS(fd)
//	//for i, file := range files {
//	//	log.Println(i)
//	//	log.Println(file.name)
//	//}
//	//转换函数测试
//
//	//var bytesTest []byte
//	//bytesTest = append(bytesTest, 3)
//	//bytesTest = append(bytesTest, 1)
//	//bytesTest = append(bytesTest, 5)
//	////00000000 00000001 00000101
//	//log.Println(bytesTest)
//	//log.Println("00000010 ",bitmapUnsetetTest(0,bytesTest))
//	//
//	//log.Println("0==",bitmapGet(0,bytesTest,7))
//	//log.Println("1==",bitmapGet(8,bytesTest,20))
//
//	////writeFile 函数测试
//	//intputstr:="writeFileTest-yvjinbo"
//	//input:=[]byte(intputstr)
//	//err = writeFile(fd, input, 'd', 103741837)
//	//if err != nil {
//	//	log.Println(err)
//	//}
//
//	//readFile函数测试
//	//临时参数 bBlockId 要传进来的
//	//
//	//var bBlockId uint32
//	//bBlockId = 103741837
//	//output := readFile(fd, 'd', bBlockId)
//	//log.Println(output)
//	//log.Println(string(output))
//
//	err = syscall.Close(fd)
//
//	if err != nil {
//		log.Fatal(err)
//	}
//}
