package main

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

//读取文件函数
func readFile(fd int, readFlag byte, bBlockId uint32) []byte {

	log.Println("开始读函数")

	//读取superblock
	var sb *superBlock
	sb = ReadSuperBlock(fd)
	devSize := sb.sDevSize
	blockSize := sb.sBlockSize
	//log.Println(devSize, blockSize)
	//log.Println(unsafe.Sizeof(superBlock{}))
	//log.Println("sizeof(struct block_info):",unsafe.Sizeof(blockExtraInfo{}))
	//log.Println("sizeof(struct block_info):",unsafe.Sizeof(blockInfo{}))
	//log.Println("BlockInfoSize 信息块部分大小:",BlockInfoSize())
	//log.Println("BlockNum 数据块的个数:",BlockNum(uintptr(devSize), uintptr(blockSize)))
	//log.Println("BitmapSize bitmap 大小:",BitmapSize(uintptr(devSize), uintptr(blockSize)))
	//log.Println("BitmapSpace bitmap 占用的512块数:",BitmapSpace(uintptr(devSize), uintptr(blockSize)))
	//log.Println("BitmapOffset bitmap 偏移量:",BitmapOffset())
	//log.Println("InfoSize block info 大小:",InfoSize(uintptr(devSize), uintptr(blockSize)))
	//log.Println("InfoSpace block info 占用的512块数:",InfoSpace(uintptr(devSize), uintptr(blockSize)))
	//log.Println("InfoOffset block info 起始偏移量:",InfoOffset(uintptr(devSize), uintptr(blockSize)))
	//log.Println("ExtraSize block extra info 大小:",ExtraSize(uintptr(devSize), uintptr(blockSize)))
	//log.Println("ExtraSpace block extra info 占用的512块数:",ExtraSpace(uintptr(devSize), uintptr(blockSize)))
	//log.Println("ExtraOffset block extra inf 起始偏移量:",ExtraOffset(uintptr(devSize), uintptr(blockSize)))
	//log.Println("DataOffset 数据块的起始偏移量:",DataOffset(uintptr(devSize), uintptr(blockSize)))

	//读取bitmap
	pBitmap := readBitmap(fd, devSize, blockSize)
	log.Println("p_bitmap:", pBitmap)
	bitSize := BlockNum(uintptr(devSize), uintptr(blockSize))
	log.Println("bit_size", bitSize)
	//******************************************************************************************
	//此处是采用转二进制字符串的方式
	////p_bitmap 转二进制[]byte
	//str := BytesToBinaryString(pBitmap)
	//log.Println("bitmapToString:",str)
	//bs := StringToBytes(str)
	//log.Println("bitmapTo[]byte:", bs)
	//
	////获取有效b_block_id列表
	//usedBlocklist := bitmapUsedblockid(bs)
	//log.Println("有效b_block_id列表,pBlocklist的长度:", usedBlocklist, len(usedBlocklist))
	//******************************************************************************************
	//以下是采用位运算的方式
	usedBlocklist := bitmapUsedBlockId(pBitmap, bitSize)
	log.Println("有效b_block_id列表,pBlocklist的长度:", usedBlocklist, len(usedBlocklist))

	//根据bitmap找到有效的b_block_id列表
	//读取blockinfo
	var bis []blockInfo
	bis = readblockInfo(fd, devSize, blockSize)
	//log.Println("bis:", bis)
	for i, bi := range bis {
		fmt.Printf("bi[%d]:bi: ", i)
		fmt.Println(bi.bCreatTime, bi.bUpdateTime, bi.bCsmLength, bi.bDataLength, bi.bBlockId)
	}

	var dataBlockId = -1

	for _, bmDataBlockId := range usedBlocklist {
		if bis[bmDataBlockId].bBlockId == bBlockId {
			dataBlockId = bmDataBlockId
			break
		}
	}
	if dataBlockId < 0 {
		log.Printf("Can not find file with bBlockId %d.\n", bBlockId)
		return nil
	}

	//根据data_block_id读取csm
	//临时参数 这是一个标志 判断是校验数据还是元数据
	//readFlag = 'c'
	var readLen uint32
	var devOffset uintptr
	if readFlag == 'c' {
		readLen = bis[dataBlockId].bCsmLength
		log.Println(readLen, readFlag)
		devOffset = DataOffset(uintptr(devSize), uintptr(blockSize)) + uintptr(uint32(dataBlockId)*blockSize)
		log.Println("devOffset", devOffset)
	} else if readFlag == 'd' {
		readLen = bis[dataBlockId].bDataLength
		log.Println(readLen, readFlag)
		devOffset = DataOffset(uintptr(devSize), uintptr(blockSize)) + uintptr(uint32(dataBlockId)*blockSize) + 2*MbSize
		log.Println("devOffset", devOffset)
	}
	output := make([]byte, readLen)
	syscall.Pread(fd, output, int64(devOffset))
	log.Println("output", output)
	return output
}

//写入文件函数
func writeFile(fd int, data []byte, writeFlag byte, bBlockId uint32) error {
	log.Println("开始写函数")
	//读取superblock
	var sb *superBlock
	sb = ReadSuperBlock(fd)
	devSize := sb.sDevSize
	blockSize := sb.sBlockSize

	//读取bitmap
	pBitmap := readBitmap(fd, devSize, blockSize)
	log.Println("p_bitmap:", pBitmap)
	bitSize := BlockNum(uintptr(devSize), uintptr(blockSize))
	log.Println("bit_size", bitSize)

	//以下是采用位运算的方式
	usedBlocklist := bitmapUsedBlockId(pBitmap, bitSize)
	log.Println("有效b_block_id列表,pBlocklist的长度:", usedBlocklist, len(usedBlocklist))

	//根据bitmap找到有效的b_block_id列表
	//读取blockinfo
	var bis []blockInfo
	bis = readblockInfo(fd, devSize, blockSize)
	//log.Println("bis:", bis)
	for i, bi := range bis {
		fmt.Printf("bi[%d]:bi: ", i)
		fmt.Println(bi.bCreatTime, bi.bUpdateTime, bi.bCsmLength, bi.bDataLength, bi.bBlockId)
	}

	var dataBlockId = -1

	for _, bmDataBlockId := range usedBlocklist {
		if bis[bmDataBlockId].bBlockId == bBlockId {
			dataBlockId = bmDataBlockId
			break
		}
	}
	if len(usedBlocklist) == int(bitSize) {
		log.Fatal("数据已经写满无法继续写入。。。")
		return nil
	} else if dataBlockId < 0 {
		log.Printf("Can not find file with bBlockId %d.\n", bBlockId)
		//availableList:=bitmapAvailableblockid(bs)
		availableList := bitmapAvailableBlockId(pBitmap, bitSize)
		log.Println("availableList:", availableList)
		//假设一次只写一个文件
		//则只需要返回一个dataBlockId
		dataBlockId = availableList[0]
	}

	//根据data_block_id读取csm
	//临时参数 这是一个标志 判断是校验数据还是元数据
	//readFlag = 'c'
	var writeLen int
	writeLen = len(data)
	var devOffset uintptr
	if writeFlag == 'c' {
		log.Println(writeLen, writeFlag)
		devOffset = DataOffset(uintptr(devSize), uintptr(blockSize)) + uintptr(uint32(dataBlockId)*blockSize)
		log.Println("devOffset", devOffset)
	} else if writeFlag == 'd' {
		log.Println(writeLen, writeFlag)
		devOffset = DataOffset(uintptr(devSize), uintptr(blockSize)) + uintptr(uint32(dataBlockId)*blockSize) + 2*MbSize
		log.Println("devOffset", devOffset)
	}
	pwrite, err := syscall.Pwrite(fd, data, int64(devOffset))
	if err != nil {
		log.Println("写入失败！")
		return err
	}
	log.Println("input_len", pwrite)

	//更改bitmap
	bitmapSet(dataBlockId, pBitmap, fd, bitSize)

	//更新block_info的信息
	var bii *blockInfo
	bii = new(blockInfo)
	bii.bBlockId = bBlockId
	if writeFlag == 'c' {
		bii.bCsmLength = uint32(len(data))
		bii.bDataLength = bis[dataBlockId].bDataLength
		log.Println("bii.bCsmLength,bii.bDataLength", bii.bCsmLength, bii.bDataLength)
	} else if writeFlag == 'd' {
		bii.bDataLength = uint32(len(data))
		bii.bCsmLength = bis[dataBlockId].bCsmLength
		log.Println("bii.bCsmLength,bii.bDataLength", bii.bCsmLength, bii.bDataLength)
	}
	offset := InfoOffset(uintptr(devSize), uintptr(blockSize)) + uintptr(dataBlockId)*unsafe.Sizeof(blockInfo{})
	log.Println("blockinfo offset:", offset)

	_, err = syscall.Pwrite(fd, blockInfoToBytes(bii), int64(offset))
	if err != nil {
		log.Fatal("block_info的信息更新失败:bBlockId", bBlockId, dataBlockId)
	}
	return nil
}

//删除文件函数
func deleteFile(fd int, bBlockId uint32) error {
	log.Println("开始删除数据块:", bBlockId)

	//读取superblock
	var sb *superBlock
	sb = ReadSuperBlock(fd)
	devSize := sb.sDevSize
	blockSize := sb.sBlockSize

	//读取bitmap
	pBitmap := readBitmap(fd, devSize, blockSize)
	log.Println("p_bitmap:", pBitmap)
	bitSize := BlockNum(uintptr(devSize), uintptr(blockSize))
	log.Println("bit_size", bitSize)

	//以下是采用位运算的方式
	usedBlocklist := bitmapUsedBlockId(pBitmap, bitSize)
	log.Println("有效b_block_id列表,pBlocklist的长度:", usedBlocklist, len(usedBlocklist))

	//根据bitmap找到有效的b_block_id列表
	//读取blockinfo
	var bis []blockInfo
	bis = readblockInfo(fd, devSize, blockSize)
	//log.Println("bis:", bis)
	for i, bi := range bis {
		fmt.Printf("bi[%d]:bi: ", i)
		fmt.Println(bi.bCreatTime, bi.bUpdateTime, bi.bCsmLength, bi.bDataLength, bi.bBlockId)
	}

	var dataBlockId = -1

	for _, bmDataBlockId := range usedBlocklist {
		if bis[bmDataBlockId].bBlockId == bBlockId {
			dataBlockId = bmDataBlockId
			break
		}
	}

	//删除操作
	//改写bitmap
	bitmapUnset(dataBlockId, pBitmap, fd, bitSize)

	err := syscall.Close(fd)
	if err != nil {
		log.Fatal(err)
	}

	return err
}

//匹配name中的bBlockId
func match(name string) (bBlockId uint32, flag byte) {
	if strings.Contains(name, "meta") {
		flag = 'c'
		path := regexp.MustCompile(`^blk_([\d]+)_([\d]+).meta$`)
		params := path.FindStringSubmatch(name)
		parseUint, err := strconv.ParseUint(params[1], 10, 32)
		if err != nil {
			log.Fatal("stringTouint32 失败!")
		}
		bBlockId = uint32(parseUint)
	} else {
		flag = 'd'
		path := regexp.MustCompile(`^blk_([\d]+)$`)
		params := path.FindStringSubmatch(name)
		parseUint, err := strconv.ParseUint(params[1], 10, 32)
		if err != nil {
			log.Fatal("stringTouint32 失败!")
		}
		bBlockId = uint32(parseUint)
	}
	return bBlockId, flag
}

//文件系统初始化函数
func startFS(fd int) (files []*File) {

	log.Println("开始更新硬盘初始状态信息...")

	//读取superblock
	var sb *superBlock
	sb = ReadSuperBlock(fd)
	devSize := sb.sDevSize
	blockSize := sb.sBlockSize

	//读取bitmap
	pBitmap := readBitmap(fd, devSize, blockSize)
	log.Println("p_bitmap:", pBitmap)
	bitSize := BlockNum(uintptr(devSize), uintptr(blockSize))
	log.Println("bit_size", bitSize)

	//以下是采用位运算的方式
	usedBlocklist := bitmapUsedBlockId(pBitmap, bitSize)
	log.Println("有效b_block_id列表,pBlocklist的长度:", usedBlocklist, len(usedBlocklist))

	//根据bitmap找到有效的b_block_id列表
	//读取blockinfo
	var bis []blockInfo
	bis = readblockInfo(fd, devSize, blockSize)
	//log.Println("bis:", bis)
	var fileNames []string

	for _, dataBlockId := range usedBlocklist {
		fileNames = append(fileNames, "blk_"+strconv.Itoa(int(bis[dataBlockId].bBlockId)))
	}

	for _, name := range fileNames {
		var file *File
		file = new(File)
		file.name = name
		files = append(files, file)
	}
	return files
}

//****************************废弃函数**************************************

// StringToBytes string转[]byte
func StringToBytes(str string) []byte {
	bs := []byte(str)
	return bs
}

// BytesToString []byte转string
func BytesToString(bs []byte) string {
	str := string(bs)
	return str
}

// BytesToBinaryString []byte转二进制字符串
func BytesToBinaryString(bs []byte) string {
	buf := bytes.NewBuffer([]byte{})

	for _, v := range bs {
		buf.WriteString(fmt.Sprintf("%b", v))
	}
	return buf.String()
}

// BinaryStringToDEC  二进制字符串转十进制int
func BinaryStringToDEC(s string) (num int) {
	l := len(s)
	for i := l - 1; i >= 0; i-- {
		num += (int(s[l-i-1]) & 0xf) << uint8(i)
		//num += (int(s[l-i-1]) - 48) << uint8(i)
	}
	return num
}
