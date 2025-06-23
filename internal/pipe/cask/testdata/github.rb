module GitHubHelper
  def self.get_asset_api_url(tag, name)
    require "utils/github"
    release = GitHub.get_release("goreleaser", "example", tag)
    release["assets"].find { |asset| asset["name"] == name }["url"]
  end

  def self.token
    require "utils/github"
    @github_token = ENV["HOMEBREW_GITHUB_API_TOKEN"]
    unless @github_token
      @github_token = GitHub::API.credentials
      raise "Failed to retrieve token" if @github_token.nil? || @github_token.empty?
    end
    @github_token
  end
end
