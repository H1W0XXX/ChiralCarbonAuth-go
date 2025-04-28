#!/usr/bin/env python3
import sys

def build_index(sdf_path: str, idx_path: str) -> None:
    """
    扫描 sdf_path，每遇到一行 "$$$$"，就将下一行的文件偏移写入 idx_path。
    """
    with open(sdf_path, 'rb') as sdf, open(idx_path, 'w', encoding='utf-8') as idx:
        while True:
            line = sdf.readline()
            if not line:
                break
            # 去除两端空白后，判断是否为 $$$$
            if line.strip() == b'$$$$':
                # f.tell() 恰好是下一行的起始字节偏移
                offset = sdf.tell()
                idx.write(f"{offset}\n")

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("用法: python build_index.py <input.sdf> <output.idx>")
        sys.exit(1)
    sdf_file = sys.argv[1]
    idx_file = sys.argv[2]
    build_index(sdf_file, idx_file)
    print(f"生成索引 {idx_file} 完毕")
