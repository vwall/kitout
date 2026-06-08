class Kitout < Formula
  desc "Declarative setup tool for fresh Macs"
  homepage "https://github.com/vwall/kitout"
  version "1.1.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/vwall/kitout/releases/download/v1.1.0/kitout_1.1.0_darwin_arm64.tar.gz"
      sha256 "699dcc8c754fb06ec95b6466aadd3c1c182b8a7900d2d518b79d9204f832d1bb"
    end

    on_intel do
      url "https://github.com/vwall/kitout/releases/download/v1.1.0/kitout_1.1.0_darwin_amd64.tar.gz"
      sha256 "15065c87a957510adfff2b9325ab716c9bce3b8a01be8cc31f766364a7be0029"
    end
  end

  def install
    bin.install "kitout"
  end

  test do
    assert_match "kitout #{version}", shell_output("#{bin}/kitout version")

    config = testpath/"kitout.yaml"
    system bin/"kitout", "init", "--config", config
    assert_match "missing", shell_output("#{bin}/kitout status --config #{config}", 1)
  end
end
