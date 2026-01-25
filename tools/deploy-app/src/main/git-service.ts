import simpleGit, { SimpleGit, StatusResult, LogResult } from 'simple-git'

export interface GitStatus {
  branch: string
  tracking: string | null
  ahead: number
  behind: number
  staged: FileChange[]
  unstaged: FileChange[]
  untracked: string[]
  lastCommit: CommitInfo | null
}

export interface FileChange {
  path: string
  status: 'modified' | 'added' | 'deleted' | 'renamed' | 'copied'
}

export interface CommitInfo {
  hash: string
  shortHash: string
  message: string
  author: string
  date: string
}

export class GitService {
  private git: SimpleGit

  constructor(repoPath: string) {
    this.git = simpleGit(repoPath)
  }

  async getStatus(): Promise<GitStatus> {
    const [status, log] = await Promise.all([
      this.git.status(),
      this.git.log({ maxCount: 1 }).catch(() => null)
    ])

    return {
      branch: status.current || 'unknown',
      tracking: status.tracking || null,
      ahead: status.ahead,
      behind: status.behind,
      staged: this.parseFileChanges(status.staged, status),
      unstaged: this.parseFileChanges(status.modified, status).concat(
        this.parseFileChanges(status.deleted, status, 'deleted')
      ),
      untracked: status.not_added,
      lastCommit: log && log.latest ? {
        hash: log.latest.hash,
        shortHash: log.latest.hash.substring(0, 7),
        message: log.latest.message,
        author: log.latest.author_name,
        date: log.latest.date
      } : null
    }
  }

  private parseFileChanges(files: string[], status: StatusResult, defaultStatus: FileChange['status'] = 'modified'): FileChange[] {
    return files.map(path => {
      let fileStatus: FileChange['status'] = defaultStatus

      if (status.created.includes(path)) {
        fileStatus = 'added'
      } else if (status.deleted.includes(path)) {
        fileStatus = 'deleted'
      } else if (status.renamed.some(r => r.to === path)) {
        fileStatus = 'renamed'
      }

      return { path, status: fileStatus }
    })
  }

  async getLog(count: number = 10): Promise<CommitInfo[]> {
    const log = await this.git.log({ maxCount: count })

    return log.all.map(commit => ({
      hash: commit.hash,
      shortHash: commit.hash.substring(0, 7),
      message: commit.message,
      author: commit.author_name,
      date: commit.date
    }))
  }

  async stageFiles(files: string[]): Promise<void> {
    await this.git.add(files)
  }

  async unstageFiles(files: string[]): Promise<void> {
    await this.git.reset(['HEAD', '--', ...files])
  }

  async stageAll(): Promise<void> {
    await this.git.add('-A')
  }

  async commit(message: string): Promise<CommitInfo> {
    const result = await this.git.commit(message)

    return {
      hash: result.commit,
      shortHash: result.commit.substring(0, 7),
      message: message,
      author: '',
      date: new Date().toISOString()
    }
  }

  async push(): Promise<void> {
    await this.git.push()
  }

  async pull(): Promise<void> {
    await this.git.pull()
  }

  async getBranch(): Promise<string> {
    const status = await this.git.status()
    return status.current || 'unknown'
  }

  async getRemoteUrl(): Promise<string | null> {
    try {
      const remotes = await this.git.getRemotes(true)
      const origin = remotes.find(r => r.name === 'origin')
      return origin?.refs?.fetch || null
    } catch {
      return null
    }
  }
}
