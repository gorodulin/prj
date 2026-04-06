class Prj < Formula
  desc "Projector: project folder, metadata, and link manager"
  homepage "https://github.com/gorodulin/prj"
  url "https://github.com/gorodulin/prj/archive/refs/tags/v0.4.0.tar.gz"
  sha256 "bda460d2dd419a1df5100677d89d56f0781355c626ff25f6b50f02b3195a859e"
  license "Apache-2.0"

  depends_on "go" => :build

  def install
    ldflags = "-s -w -X github.com/gorodulin/prj/cmd.version=#{version}"
    system "go", "build", "-trimpath", *std_go_args(ldflags:)
    generate_completions_from_executable(bin/"prj", "completion")
  end

  def caveats
    <<~EOS
      Create a config file to get started:

        mkdir -p "#{Dir.home}/Library/Application Support/prj"

      Then write a minimal config:

        echo '{ "projects_folder": "/path/to/your/projects" }' > \\
          "#{Dir.home}/Library/Application Support/prj/config.json"

      See `prj --help` for available commands.
    EOS
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/prj --version")
  end
end
