
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package options

const (
	RSAKey     = "rsa"
	Ed25519Key = "ed25519"

	DefaultRSALen = 2048
)

type KeyGenerateSettings struct {
	Algorithm string
	Size      int
}

type KeyRenameSettings struct {
	Force bool
}

type KeyGenerateOption func(*KeyGenerateSettings) error
type KeyRenameOption func(*KeyRenameSettings) error

func KeyGenerateOptions(opts ...KeyGenerateOption) (*KeyGenerateSettings, error) {
	options := &KeyGenerateSettings{
		Algorithm: RSAKey,
		Size:      -1,
	}

	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil, err
		}
	}
	return options, nil
}

func KeyRenameOptions(opts ...KeyRenameOption) (*KeyRenameSettings, error) {
	options := &KeyRenameSettings{
		Force: false,
	}

	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil, err
		}
	}
	return options, nil
}

type keyOpts struct{}

var Key keyOpts

//类型是key.generate的选项，用于指定
//应该用于键。默认为Options.rsakey
//
//支持的密钥类型：
//*选项.rsakey
//*选项.ed25519键
func (keyOpts) Type(algorithm string) KeyGenerateOption {
	return func(settings *KeyGenerateSettings) error {
		settings.Algorithm = algorithm
		return nil
	}
}

//大小是键的选项。Generate指定键的大小
//生成。缺省值为-1
//
//值-1表示“使用密钥类型的默认大小”：
//* 2048用于RSA
func (keyOpts) Size(size int) KeyGenerateOption {
	return func(settings *KeyGenerateSettings) error {
		settings.Size = size
		return nil
	}
}

//force是key.rename的选项，用于指定是否允许
//替换现有密钥。
func (keyOpts) Force(force bool) KeyRenameOption {
	return func(settings *KeyRenameSettings) error {
		settings.Force = force
		return nil
	}
}
