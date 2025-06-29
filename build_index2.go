package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const WorkerCount = 24           // 并发 worker 数
const Timeout = 10 * time.Second // 超时限制

type task struct {
	Offset int64
	MolStr string
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("用法: build_index <input.sdf> <output.index>")
		os.Exit(1)
	}
	if err := buildIndexParallel(os.Args[1], os.Args[2]); err != nil {
		fmt.Println("生成索引失败:", err)
		os.Exit(1)
	}
	fmt.Println("索引生成完毕:", os.Args[2])
}

func buildIndexParallel(sdfPath, idxPath string) error {
	// 加载进度
	resumeProcOffset := int64(0)
	if pf, err := os.Open("progress.log"); err == nil {
		scanner := bufio.NewScanner(pf)
		for scanner.Scan() {
			if off, err := strconv.ParseInt(scanner.Text(), 10, 64); err == nil {
				resumeProcOffset = off
			}
		}
		pf.Close()
		fmt.Printf("从 progress.log 恢复，已处理到偏移 %d\n", resumeProcOffset)
	}

	// 打开输入文件
	inFile, err := os.Open(sdfPath)
	if err != nil {
		return err
	}
	defer inFile.Close()

	// 打开输出文件
	outFile, err := os.Create(idxPath)
	if err != nil {
		return err
	}
	defer outFile.Close()
	writer := bufio.NewWriter(outFile)
	defer writer.Flush()

	// 设置并发
	runtime.GOMAXPROCS(WorkerCount)

	taskCh := make(chan task, WorkerCount*4)
	resultCh := make(chan int64, WorkerCount*4)

	var wg sync.WaitGroup

	// writer goroutine
	var writeWg sync.WaitGroup
	writeWg.Add(1)
	go func() {
		defer writeWg.Done()
		for off := range resultCh {
			fmt.Fprintf(writer, "%d\n", off)
		}
	}()

	// worker pool
	for i := 0; i < WorkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range taskCh {
				// 设置超时处理
				done := make(chan struct{})
				go func() {
					defer close(done)
					mol, err := ParseMolString(t.MolStr)
					if err == nil {
						Hydrogenate(mol)
						if len(GetMoleculeChiralCarbons(mol)) >= 3 {
							resultCh <- t.Offset
						}
					}
				}()
				select {
				case <-done:
					// 正常完成
				case <-time.After(Timeout):
					// 超时，跳过当前分子
					fmt.Printf("分子 offset=%d 处理超时，自动跳过\n", t.Offset)
				}
			}
		}()
	}

	// 读取 SDF 文件并派发任务
	reader := bufio.NewReader(inFile)
	var molStart, offset int64
	var molBuf strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}
		offset += int64(len(line))
		if strings.TrimRight(line, "\r\n") == "$$$$" {
			//if strings.TrimSpace(line) == "$$$$" {
			// 记录分子进度
			if molStart > resumeProcOffset {
				taskCh <- task{
					Offset: molStart,
					MolStr: molBuf.String(),
				}
			}
			// 写入进度文件
			progressLine := fmt.Sprintf("%d\n", molStart)
			if err := writeProgress("progress.log", progressLine); err != nil {
				fmt.Println("写进度失败，跳过：", molStart, err)
			}

			// 下一个分子
			molBuf.Reset()
			molStart = offset
		} else {
			molBuf.WriteString(line)
		}

		if err == io.EOF {
			break
		}
	}

	// 收尾处理
	close(taskCh)
	wg.Wait()
	close(resultCh)
	writeWg.Wait()

	return nil
}

func writeProgress(filename, progressLine string) error {
	progFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer progFile.Close()

	_, err = progFile.WriteString(progressLine)
	if err != nil {
		return err
	}

	// 刷新数据到磁盘
	if err := progFile.Sync(); err != nil {
		return err
	}

	return nil
}

// 把这个文件改成main.go编译建立索引
// go build -o z:\build_index.exe main.go handler.go render_molecule.go sdf.go types.go utils.go chiral.go
