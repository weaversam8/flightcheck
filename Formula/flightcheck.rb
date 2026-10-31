class Flightcheck < Formula
  desc "Keep hand-coding skills current while working with Claude Code"
  homepage "https://github.com/weaversam8/flightcheck"
  # scripts/release.sh adds `url` and `sha256` here for each tagged release.
  # Until the first release, install with --HEAD.
  license "MIT"
  head "https://github.com/weaversam8/flightcheck.git", branch: "main"

  depends_on "go" => :build

  def install
    ldflags = "-s -w -X main.version=#{version}"
    system "go", "build", *std_go_args(ldflags:, output: bin/"flightcheck"), "./cmd/flchk"
    bin.install_symlink "flightcheck" => "flchk"
  end

  def caveats
    <<~EOS
      Add flightcheck's hooks to Claude Code with:
        flightcheck install

      It is also installed under the short name `flchk`.
    EOS
  end

  test do
    assert_match "flightcheck", shell_output("#{bin}/flightcheck version")
    (testpath/"settings.json").write("{}")
    ENV["FLIGHTCHECK_HOME"] = testpath/"flightcheck-data"
    system bin/"flightcheck", "install", "--settings", testpath/"settings.json"
    assert_match "hook pre-tool-use", (testpath/"settings.json").read
  end
end
