class Prj < Formula
  desc "Projector: project folder, metadata, and link manager"
  homepage "https://github.com/gorodulin/prj"
  url "https://github.com/gorodulin/prj/archive/refs/tags/v0.13.0.tar.gz"
  sha256 "8666b6983a3053da1476b4b629bcaa591cae438be7a9e03cda87dd4bd40f4c46"
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
