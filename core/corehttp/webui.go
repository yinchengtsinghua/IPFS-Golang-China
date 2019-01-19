
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package corehttp

//TODO:移动到IPN
const WebUIPath = "/ipfs/QmXc9raDM1M5G5fpBnVyQ71vR4gbnskwnB9iMEzBuLgvoZ"

//这是所有过去的WebUI路径的列表。
var WebUIPaths = []string{
	WebUIPath,
	"/ipfs/QmenEBWcAk3tN94fSKpKFtUMwty1qNwSYw3DMDFV6cPBXA",
	"/ipfs/QmUnXcWZC5Ve21gUseouJsH5mLAyz5JPp8aHsg8qVUUK8e",
	"/ipfs/QmSDgpiHco5yXdyVTfhKxr3aiJ82ynz8V14QcGKicM3rVh",
	"/ipfs/QmRuvWJz1Fc8B9cTsAYANHTXqGmKR9DVfY5nvMD1uA2WQ8",
	"/ipfs/QmQLXHs7K98JNQdWrBB2cQLJahPhmupbDjRuH1b9ibmwVa",
	"/ipfs/QmXX7YRpU7nNBKfw75VG7Y1c3GwpSAGHRev67XVPgZFv9R",
	"/ipfs/QmXdu7HWdV6CUaUabd9q2ZeA4iHZLVyDRj3Gi4dsJsWjbr",
	"/ipfs/QmaaqrHyAQm7gALkRW8DcfGX3u8q9rWKnxEMmf7m9z515w",
	"/ipfs/QmSHDxWsMPuJQKWmVA1rB5a3NX2Eme5fPqNb63qwaqiqSp",
	"/ipfs/QmctngrQAt9fjpQUZr7Bx3BsXUcif52eZGTizWhvcShsjz",
	"/ipfs/QmS2HL9v5YeKgQkkWMvs1EMnFtUowTEdFfSSeMT4pos1e6",
	"/ipfs/QmR9MzChjp1MdFWik7NjEjqKQMzVmBkdK3dz14A6B5Cupm",
	"/ipfs/QmRyWyKWmphamkMRnJVjUTzSFSAAZowYP4rnbgnfMXC9Mr",
	"/ipfs/QmU3o9bvfenhTKhxUakbYrLDnZU7HezAVxPM6Ehjw9Xjqy",
	"/ipfs/QmPhnvn747LqwPYMJmQVorMaGbMSgA7mRRoyyZYz3DoZRQ",
}

var WebUIOption = RedirectOption("webui", WebUIPath)
