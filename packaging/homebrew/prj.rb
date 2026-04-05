class Prj < Formula
  desc "CLI tool for managing project folders, metadata, and links"
  homepage "https://github.com/gorodulin/prj"
  url "https://github.com/gorodulin/prj/archive/refs/tags/v0.2.0.tar.gz"
  sha256 "0d1df18fdf059f1f57f28da256c9bd20b39e8fc1f07637adb8d0d02df61a0b7a"
  license "Apache-2.0"

  depends_on "go" => :build

  def install
    ldflags = "-s -w -X github.com/gorodulin/prj/cmd.version=#{version}"
    system "go", "build", *std_go_args(ldflags:)
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
