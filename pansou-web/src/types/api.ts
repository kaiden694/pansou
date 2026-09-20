export interface PanLink {
  type: string
  url: string
  password?: string
  datetime?: string
  work_title?: string
}

export interface SearchResult {
  message_id: string
  unique_id: string
  channel: string
  datetime: string
  title: string
  content: string
  links: PanLink[]
  tags?: string[]
  images?: string[]
}

export interface MergedLink {
  url: string
  password?: string
  note?: string
  datetime?: string
  source?: string
  images?: string[]
}

export interface SearchResponse {
  total: number
  results: SearchResult[]
  merged_by_type: Record<string, MergedLink[]>
}

export interface HealthResponse {
  status: string
  auth_enabled: boolean
  plugins_enabled: boolean
  plugin_count?: number
  plugins?: string[]
  channels: string[]
  channels_count: number
}
