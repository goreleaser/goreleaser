module GitHubHelper
  def self.github_token
    require "utils/github"

    token = ENV["HOMEBREW_GITHUB_API_TOKEN"]
    token ||= GitHub::API.credentials
    raise "Failed to retrieve github api token" if token.nil? || token.empty?

    token
  end

  def self.get_asset_api_url(tag, name)
    require "json"
    require "net/http"
    require "uri"

    resp = Net::HTTP.get(
      URI.parse("https://api.github.com/repos/goreleaser/example/releases/tags/#{tag}"),
      {
        "Accept" => "application/vnd.github+json",
        "Authorization" => "Bearer #{github_token}",
        "X-GitHub-Api-Version" => "2022-11-28"
      }
    )

    release = JSON.parse(resp)
    release["assets"].find { |asset| asset["name"] == name }["url"]
  end
end
