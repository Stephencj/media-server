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

// API response wrapper
struct SectionsResponse: Codable {
    let sections: [LibrarySection]
}

// LibrarySection with media items
struct SectionWithMedia: Codable {
    let section: LibrarySection
    let media: [Media]
}
