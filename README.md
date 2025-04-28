
# ChiralCarbonAuth

本项目基于重写 [cinit/NeoAuthBotPlugin](https://github.com/cinit/NeoAuthBotPlugin)，用于手性碳分子认证。

## 功能简介

- 从 PubChem 数据库中随机抽取一个小分子。
- 提供网页接口渲染分子结构。
- 用于生成手性碳识别验证题目。

## 使用方法

### 1. 下载 SDF 数据

从 PubChem 官方 FTP 下载 `.sdf` 文件：

```
https://ftp.ncbi.nlm.nih.gov/pubchem/Compound/CURRENT-Full/SDF/
```

例如下载 `Compound_156500001_157000000.sdf`。

### 2. 编译索引生成器

在项目根目录执行：

```bash
go build -o z:\0\build_index.exe build_index.go chiral.go sdf.go types.go utils.go
```

生成 `build_index.exe`。

### 3. 生成索引文件

使用生成的工具建立 `.index` 文件：

```bash
.\build_index.exe Compound_156500001_157000000.sdf Compound_156500001_157000000.index
```

### 4. 修改源码配置

打开 `handler.go`，找到并修改以下行：

```go
mol, err = pickRandomMoleculeFromIndexed("Compound_156500001_157000000.sdf", "Compound_156500001_157000000.index")
```

替换为你自己的 `.sdf` 和 `.index` 文件名。

### 5. 修改端口号（可选）

打开 `main.go`，找到：

```go
http.ListenAndServe(":8080", nil)
```

可以将端口号改为需要的值，例如 `27419`。

### 6. 去除星号提示（可选）

如果想去掉网页中的星号提示，打开 `render_molecule.go`，修改相关渲染逻辑。

### 7. 运行项目

运行服务器：

```bash
go run main.go handler.go render_molecule.go sdf.go types.go utils.go chiral.go
```

或先编译：

```bash
go build -o auth_server.exe main.go handler.go render_molecule.go sdf.go types.go utils.go chiral.go
.\auth_server.exe
```

访问浏览器：

```
http://127.0.0.1:8080
```

## 注意事项

- `.sdf` 和 `.index` 文件需要在正确路径下，或使用绝对路径。
- `.sdf` 文件较大，建议选用部分数据进行测试，解压后的文件5-10g。
- 部署到服务器时需开放对应端口。

