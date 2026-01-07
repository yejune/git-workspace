class GitSubclone < Formula
  desc "Manage nested git repositories with independent push capability"
  homepage "https://github.com/yejune/git-subclone"
  url "https://github.com/yejune/git-subclone/archive/refs/tags/v0.0.9.tar.gz"
  sha256 "1d2ed3996efc4b4ea80d762866eebb526e483704ab81fcc8b542bfa0ceb1d8ad"
  license "MIT"
  head "https://github.com/yejune/git-subclone.git", branch: "main"

  depends_on "go" => :build

  def install
        ldflags = "-X github.com/yejune/git-subclone/cmd.Version=#{version}"
    system "go", "build", "-ldflags", ldflags, "-o", "git-subclone", "."
    bin.install "git-subclone"
  end

  def test
        assert_match "version", shell_output("#{bin}/git-subclone --version")
  end

  def caveats
    <<~EOS
            git-subclone has been installed!
      
      Quick Start:
        git subclone https://github.com/user/repo.git
        git subclone https://github.com/user/repo.git packages/repo
        git subclone -b develop https://github.com/user/repo.git
      
      Commands:
        git subclone sync      Sync all subclones
        git subclone list      List subclones
        git subclone push      Push subclone(s)
        git subclone status    Show status
    EOS
  end
end
