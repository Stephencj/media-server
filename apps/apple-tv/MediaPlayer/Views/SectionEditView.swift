import SwiftUI

struct SectionEditView: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel: SectionEditViewModel

    let section: Section?

    init(section: Section? = nil) {
        self.section = section
        _viewModel = StateObject(wrappedValue: SectionEditViewModel(section: section))
    }

    var body: some View {
        Form {
            Section("Basic Information") {
                HStack {
                    Text("Name")
                        .frame(width: 150, alignment: .trailing)
                    TextField("Section Name", text: $viewModel.name)
                        .textFieldStyle(.roundedBorder)
                }

                HStack {
                    Text("Icon")
                        .frame(width: 150, alignment: .trailing)
                    TextField("film, tv, star...", text: $viewModel.icon)
                        .textFieldStyle(.roundedBorder)
                }

                HStack {
                    Text("Description")
                        .frame(width: 150, alignment: .trailing)
                    TextField("Optional description", text: $viewModel.description)
                        .textFieldStyle(.roundedBorder)
                }
            }

            Section("Settings") {
                HStack {
                    Text("Type")
                        .frame(width: 150, alignment: .trailing)
                    Picker("", selection: $viewModel.sectionType) {
                        Text("Smart").tag(SectionType.smart)
                        Text("Standard").tag(SectionType.standard)
                        Text("Folder").tag(SectionType.folder)
                    }
                    .pickerStyle(.segmented)
                }

                HStack {
                    Text("Display Order")
                        .frame(width: 150, alignment: .trailing)
                    Stepper("\(viewModel.displayOrder)", value: $viewModel.displayOrder, in: 0...100)
                }

                Toggle("Visible", isOn: $viewModel.isVisible)
            }

            if viewModel.sectionType == .smart {
                Section("Rules") {
                    ForEach(viewModel.rules) { rule in
                        HStack {
                            Text(rule.field)
                            Text(rule.operator)
                            Text(rule.value)
                            Spacer()
                            Button("Delete") {
                                viewModel.deleteRule(rule)
                            }
                        }
                    }

                    Button("Add Rule") {
                        viewModel.showRuleBuilder = true
                    }
                }
            }

            Section {
                HStack {
                    Spacer()
                    Button("Cancel") {
                        dismiss()
                    }
                    .buttonStyle(.bordered)

                    Button(section == nil ? "Create" : "Update") {
                        Task {
                            await viewModel.save()
                            dismiss()
                        }
                    }
                    .buttonStyle(.borderedProminent)
                    Spacer()
                }
            }
        }
        .navigationTitle(section == nil ? "Create Section" : "Edit Section")
        .sheet(isPresented: $viewModel.showRuleBuilder) {
            RuleBuilderView { rule in
                viewModel.addRule(rule)
            }
        }
        .alert("Error", isPresented: $viewModel.showError) {
            Button("OK", role: .cancel) { }
        } message: {
            Text(viewModel.errorMessage)
        }
    }
}

// ViewModel
@MainActor
class SectionEditViewModel: ObservableObject {
    @Published var name = ""
    @Published var icon = ""
    @Published var description = ""
    @Published var sectionType: SectionType = .smart
    @Published var displayOrder = 0
    @Published var isVisible = true
    @Published var rules: [SectionRule] = []
    @Published var showRuleBuilder = false
    @Published var showError = false
    @Published var errorMessage = ""

    private let section: Section?
    private let apiClient = APIClient.shared

    init(section: Section?) {
        self.section = section
        if let section = section {
            self.name = section.name
            self.icon = section.icon
            self.description = section.description ?? ""
            self.sectionType = section.sectionType
            self.displayOrder = section.displayOrder
            self.isVisible = section.isVisible
            self.rules = section.rules ?? []
        }
    }

    func addRule(_ rule: SectionRule) {
        rules.append(rule)
    }

    func deleteRule(_ rule: SectionRule) {
        rules.removeAll { $0.id == rule.id }
    }

    func save() async {
        // Implement save logic via API
        // This would call createSection or updateSection
    }
}

// Rule Builder View
struct RuleBuilderView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var field = "type"
    @State private var op = "equals"
    @State private var value = ""

    let onAdd: (SectionRule) -> Void

    var body: some View {
        Form {
            Picker("Field", selection: $field) {
                Text("Type").tag("type")
                Text("Genre").tag("genres")
                Text("Year").tag("year")
                Text("Rating").tag("rating")
                Text("Resolution").tag("resolution")
            }

            Picker("Operator", selection: $op) {
                Text("Equals").tag("equals")
                Text("Contains").tag("contains")
                Text("Greater Than").tag("greater_than")
                Text("Less Than").tag("less_than")
                Text("In Range").tag("in_range")
            }

            TextField("Value", text: $value)

            HStack {
                Spacer()
                Button("Cancel") {
                    dismiss()
                }
                .buttonStyle(.bordered)

                Button("Add") {
                    let rule = SectionRule(
                        id: 0,
                        sectionId: 0,
                        field: field,
                        operator: op,
                        value: value
                    )
                    onAdd(rule)
                    dismiss()
                }
                .buttonStyle(.borderedProminent)
                Spacer()
            }
        }
        .navigationTitle("Add Rule")
    }
}
