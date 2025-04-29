import os

def read_index_file(index_file):
    """读取索引文件，返回所有的偏移量"""
    offsets = []
    with open(index_file, 'r') as f:
        for line in f:
            try:
                offsets.append(int(line.strip()))
            except ValueError:
                continue
    return offsets

def read_sdf_by_offset(sdf_file, offsets):
    """根据索引文件中的偏移量，读取对应的分子数据"""
    molecules = []
    with open(sdf_file, 'r') as f:
        for offset in offsets:
            f.seek(offset)
            mol_str = ''
            while True:
                line = f.readline()
                if not line:
                    break
                mol_str += line
                if "$$$$" in line:  # SDF 文件的分子结束标志
                    break
            if mol_str:
                molecules.append(mol_str)
    return molecules

def filter_molecules(molecules):
    """筛选符合条件的分子"""
    filtered_molecules = []
    for mol in molecules:
        if 'C' in mol and len(mol) > 100:  # 示例条件：含碳原子且分子较大
            filtered_molecules.append(mol)
    return filtered_molecules

def write_filtered_molecules(output_sdf_file, filtered_molecules):
    """将筛选后的分子写入新的 SDF 文件，并返回它们的偏移量"""
    offsets = []
    with open(output_sdf_file, 'w') as f:
        for mol in filtered_molecules:
            offset = f.tell()  # 记录当前分子在文件中的起始位置
            f.write(mol)       # 写入分子（已包含 $$$$ 分隔符）
            offsets.append(offset)
    return offsets

def write_new_index(output_index_file, offsets):
    """更新新的索引文件"""
    with open(output_index_file, 'w') as f:
        for offset in offsets:
            f.write(f"{offset}\n")

def merge_sdfs(index_files, output_sdf_file, output_index_file, input_sdf_files):
    all_molecules = []

    # 读取所有输入 SDF 文件的分子数据
    for idx, sdf_file in zip(index_files, input_sdf_files):
        print(f"处理文件：{sdf_file}")
        offsets = read_index_file(idx)  # 获取索引文件中的偏移量
        molecules = read_sdf_by_offset(sdf_file, offsets)  # 根据偏移量读取分子
        all_molecules.extend(molecules)

    # 筛选符合条件的分子
    filtered_molecules = filter_molecules(all_molecules)

    # 将筛选后的分子写入新的 SDF 文件，并获取新偏移量
    new_offsets = write_filtered_molecules(output_sdf_file, filtered_molecules)

    # 使用新偏移量更新索引文件
    write_new_index(output_index_file, new_offsets)

def main():
    # 定义输入的 SDF 文件和索引文件
    input_sdf_files = [
        'Compound_000000001_000500000.sdf',
        'Compound_003000001_003500000.sdf',
        'Compound_156500001_157000000.sdf'
    ]
    index_files = [
        'Compound_000000001_000500000.index',
        'Compound_003000001_003500000.index',
        'Compound_156500001_157000000.index'
    ]
    output_sdf_file = 'output.sdf'
    output_index_file = 'output.index'

    merge_sdfs(index_files, output_sdf_file, output_index_file, input_sdf_files)

if __name__ == "__main__":
    main()
