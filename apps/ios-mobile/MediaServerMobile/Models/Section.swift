import Foundation

// LibrarySection represents a library section (Movies, TV Shows, or custom)
struct LibrarySection: Identifiable, Codable {
    let id: Int64
    let name: String
    let slug: String
    let icon: String
    let description: String?
    let sectionType: SectionType
    let displayOrder: Int
    let isVisible: Bool
    let mediaCount: Int?
    let rules: [SectionRule]?

    enum CodingKeys: String, CodingKey {
        case id, name, slug, icon, description, rules
        case sectionType = "section_type"
        case displayOrder = "display_order"
        case isVisible = "is_visible"
        case mediaCount = "media_count"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(Int64.self, forKey: .id)
        name = try container.decode(String.self, forKey: .name)
        slug = try container.decode(String.self, forKey: .slug)
        // Decode icon leniently: missing or null -> ""
        icon = try container.decodeIfPresent(String.self, forKey: .icon) ?? ""
        description = try container.decodeIfPresent(String.self, forKey: .description)
        sectionType = try container.decode(SectionType.self, forKey: .sectionType)
        displayOrder = try container.decode(Int.self, forKey: .displayOrder)
        isVisible = try container.decode(Bool.self, forKey: .isVisible)
        mediaCount = try container.decodeIfPresent(Int.self, forKey: .mediaCount)
        rules = try container.decodeIfPresent([SectionRule].self, forKey: .rules)
    }
}

// Section types
enum SectionType: String, Codable {
    case standard
    case smart
    case folder
}

// Section rule for smart sections
struct SectionRule: Identifiable, Codable {
    let id: Int64
    let sectionId: Int64
    let field: String
    let `operator`: String  // Using backticks because operator is a keyword
    let value: String

    enum CodingKeys: String, CodingKey {
        case id
        case sectionId = "section_id"
        case field
        case `operator` = "operator"
        case value
    }
}

// API response wrapper with lenient decoding
struct SectionsResponse: Codable {
    let sections: [LibrarySection]

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        // Decode sections leniently: missing or null -> []
        sections = try container.decodeIfPresent([LibrarySection].self, forKey: .sections) ?? []
    }
}

// LibrarySection with media items
struct SectionWithMedia: Codable {
    let section: LibrarySection
    let media: [Media]
}
