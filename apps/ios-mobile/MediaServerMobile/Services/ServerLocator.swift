import Foundation

/// Finds the shortest way to the server.
///
/// The server's public address goes out to the internet and back in through a
/// tunnel, even from the same room. That is the only way in from elsewhere,
/// but at home it sends every stream through the house's upload link twice
/// over, and several streams at once will not fit. On the home network the
/// apps talk to the server directly instead.
@MainActor
enum ServerLocator {
    /// Where the server answers on the home network.
    static let localCandidates = [
        "http://192.168.4.46:8080",
        "http://pop-os.local:8080",
    ]

    private static var nextCheck = Date.distantPast
    private static var inFlight: Task<Void, Never>?
    private static let knownServerKey = "serverLocator.knownServer"

    /// Points `AppState.serverURL` at the local address when the server is
    /// reachable there and at the public one otherwise. Cheap to call often:
    /// it looks again only every couple of minutes.
    static func choose() async {
        if let inFlight {
            await inFlight.value
            return
        }
        guard Date() >= nextCheck else { return }

        let task = Task { await probe() }
        inFlight = task
        await task.value
        inFlight = nil
    }

    private static func probe() async {
        // Ask every local address at once; off the home network they all
        // time out together rather than one after another.
        let answers = await withTaskGroup(of: (String, String?).self) { group -> [String: String] in
            for candidate in localCandidates {
                group.addTask { (candidate, await identity(of: candidate, timeout: 1.5)) }
            }
            var found: [String: String] = [:]
            for await (candidate, identity) in group {
                if let identity { found[candidate] = identity }
            }
            return found
        }

        for candidate in localCandidates {
            guard let local = answers[candidate] else { continue }
            if await isOurServer(local) {
                use(candidate)
                // Often enough to notice leaving the house.
                nextCheck = Date().addingTimeInterval(120)
                return
            }
        }

        use(Secrets.serverURL)
        // Away from home there is nothing to find, so look less often.
        nextCheck = Date().addingTimeInterval(600)
    }

    /// Something answering at a home-network address is not necessarily this
    /// server: on another network the same address may be someone else's
    /// machine, and it must not be handed the sign-in token. It is trusted
    /// only if the public address reports the same server, or, with no
    /// internet to ask, if it is the server last confirmed that way.
    private static func isOurServer(_ local: String) async -> Bool {
        if let viaPublic = await identity(of: Secrets.serverURL, timeout: 5) {
            guard viaPublic == local else { return false }
            UserDefaults.standard.set(local, forKey: knownServerKey)
            return true
        }
        return UserDefaults.standard.string(forKey: knownServerKey) == local
    }

    private static func use(_ serverURL: String) {
        if AppState.shared.serverURL != serverURL {
            print("[ServerLocator] Using \(serverURL)")
            AppState.shared.serverURL = serverURL
        }
    }

    private struct Status: Decodable {
        let containerId: String?
        let hostname: String?

        enum CodingKeys: String, CodingKey {
            case hostname
            case containerId = "container_id"
        }
    }

    /// What the server at an address says it is, or nil if nothing answers
    /// in time.
    private nonisolated static func identity(of serverURL: String, timeout: TimeInterval) async -> String? {
        guard let url = URL(string: serverURL + "/api/deploy/status") else { return nil }
        var request = URLRequest(url: url)
        request.timeoutInterval = timeout
        request.cachePolicy = .reloadIgnoringLocalCacheData

        guard let (data, response) = try? await URLSession.shared.data(for: request),
              (response as? HTTPURLResponse)?.statusCode == 200,
              let status = try? JSONDecoder().decode(Status.self, from: data) else {
            return nil
        }
        let identity = "\(status.hostname ?? "")|\(status.containerId ?? "")"
        return identity == "|" ? nil : identity
    }
}
