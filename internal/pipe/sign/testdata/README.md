# Creating test keys for GnuPG

The unit tests needs a test key to work with. I have tried to create a test keyring
on the fly and while that worked I was not able to successfully sign with that.
gpg would bail with an ioctl error which I didn't track down since using a static
key works.

This uses the `--homedir .` option to create the test keys so that we do not touch
the local keyring file.

## 1.  Create signing keys

```sh
cd $GOPATH/src/github.com/goreleaser/goreleaser/internal/pipeline/sign/testdata/gnupg
gpg --homedir . --quick-generate-key --batch --passphrase '' nopass default default 10y
```

## 2. Check that the key exists

```sh
gpg --homedir . --list-keys
pub rsa2048 2017-12-13 [SC][expires: 2027-12-11]
FB6BEDFCECE1761EDD68BF32EF2D274B0EDAAE12
uid [ultimate] nopass
sub rsa2048 2017-12-13 [E]
```

## 3. Check that signing works

### 3.1 create a test file

```sh
echo "bar" > foo
```

### 3.2. sign and verfiy

```sh
gpg --homedir . --detach-sign foo
gpg --homedir . --verify foo.sig foo

gpg: Signature made Wed Dec 13 22:02:49 2017 CET
gpg: using RSA key FB6BEDFCECE1761EDD68BF32EF2D274B0EDAAE12
gpg: Good signature from "nopass" [ultimate]
```

### 3.3. cleanup

```sh
rm foo foo.sig
```

## 4. Make sure you have keyrings for both gpg1 and gpg2

travis-ci.org runs on an old Ubuntu installation which
has gpg 1.4 installed. We need to provide keyrings that
have the same keys and users for both formats.

This demonstrates the conversion from gpg2 to gpg1
format but should work the same the other way around.

### 4.1. get gpg version

```sh
gpg --version
gpg (GnuPG) 2.2.3
```

### 4.2. install gpg1

```sh
brew install gpg1
# brew install gpg2 # if you have gpg1 installed
```

### 4.3. migrate the keys from gpg2 to gpg1

```sh
gpg --homedir . --export nopass | gpg1 --homedir . --import
gpg --homedir . --export-secret-key nopass | gpg1 --homedir . --import
```

### 4.4. check keys are the same

```sh
gpg --homedir . --list-keys --keyid-format LONG
gpg1 --homedir . --list-keys --keyid-format LONG

gpg --homedir . --list-secret-keys --keyid-format LONG
gpg1 --homedir . --list-secret-keys --keyid-format LONG
```
