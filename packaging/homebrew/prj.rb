class Prj < Formula
  desc "Projector: project folder, metadata, and link manager"
  homepage "https://github.com/gorodulin/prj"
  url "https://github.com/gorodulin/prj/archive/refs/tags/v0.5.0.tar.gz"
  sha256 "8a5c17fb7b9f52fb6b59f7988c4f427565841d74b7130c02514be798500763d9"
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
