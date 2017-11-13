class Test < Formula
  desc "Some desc"
  homepage "https://google.com"
  url "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Darwin_x86_64.tar.gz"
  version "0.1.3"
  sha256 "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c68"
  
  depends_on "gtk+"
  
  conflicts_with "svn"

  def install
    custom install script
    another install script
  end

  def caveats
    "Here are some caveats"
  end

  plist_options :startup => false

  def plist; <<-EOS.undent
    it works
    EOS
  end

  test do
    system "#{bin}/foo -version"
  end
end
