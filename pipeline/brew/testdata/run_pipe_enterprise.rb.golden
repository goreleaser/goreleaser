class RunPipe < Formula
  desc "A run pipe test formula"
  homepage "https://github.com/goreleaser"
  url "http://github.example.org/test/test/releases/download/v1.0.1/bin.tar.gz"
  version "1.0.1"
  sha256 "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
  
  depends_on "zsh"
  depends_on "bash"
  
  conflicts_with "gtk+"
  conflicts_with "qt"

  def install
    bin.install "foo"
  end

  def caveats
    "don't do this"
  end

  plist_options :startup => false

  def plist; <<-EOS.undent
    <xml>whatever</xml>
    EOS
  end

  test do
    system "true"
    system "#{bin}/foo -h"
  end
end
