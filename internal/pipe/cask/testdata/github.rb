module GitHubHelper
  def self.token
    require "utils/github"

    # Prefer environment variable if available
    github_token = ENV["HOMEBREW_GITHUB_API_TOKEN"]
    github_token ||= GitHub::API.credentials
    raise "Failed to retrieve github api token" if github_token.nil? || github_token.empty?

    github_token
  end

  def self.release_asset_url(tag, name)
    require "json"
    require "net/http"
    require "uri"

    resp = Net::HTTP.get(
      # Replace with your GitHub repository URL
      URI.parse("https://api.github.com/repos/goreleaser/example/releases/tags/#{tag}"),
      {
        "Accept" => "application/vnd.github+json",
        "Authorization" => "Bearer #{token}",
        "X-GitHub-Api-Version" => "2022-11-28"
      }
    )

    release = JSON.parse(resp)
    release["assets"].find { |asset| asset["name"] == name }["url"]
  end
end
