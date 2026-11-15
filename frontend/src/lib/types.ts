// Tên ngắn cho các schema hay dùng của API (sinh trong ./api/*.d.ts)
import type { components as Admin } from './api/administration'
import type { components as Content } from './api/content'
import type { components as Identity } from './api/identity'
import type { components as Moderation } from './api/moderation'
import type { components as Notifications } from './api/notifications'
import type { components as Penalties } from './api/penalties'
import type { components as Screening } from './api/screening'

export type Me = Identity['schemas']['Me']
export type Account = Identity['schemas']['Account']
export type Role = Identity['schemas']['Role']

export type Channel = Content['schemas']['Channel']
export type Post = Content['schemas']['Post']
export type Comment = Content['schemas']['Comment']
export type Thread = Content['schemas']['Thread']
export type Feed = Content['schemas']['Feed']
export type Submitted = Content['schemas']['Submitted']
export type MyPending = Content['schemas']['MyPending']

export type Rule = Screening['schemas']['Rule']
export type CheckResult = Screening['schemas']['CheckResult']
export type DraftWarning = Screening['schemas']['Warning']
export type BlockEntry = Screening['schemas']['BlockEntry']
export type ScreeningSettings = Screening['schemas']['ScreeningSettings']
export type ScreeningStatus = Screening['schemas']['ScreeningStatus']

export type Queue = Moderation['schemas']['Queue']
export type QueueItem = Moderation['schemas']['ContentItem']
export type QueueAppeal = Moderation['schemas']['QueueAppeal']
export type MyDecision = Moderation['schemas']['MyDecision']
export type AuditEntry = Moderation['schemas']['AuditEntry']

export type Standing = Penalties['schemas']['Standing']
export type PenaltySettings = Penalties['schemas']['PenaltySettings']
export type ReadOnlyAccount = Penalties['schemas']['ReadOnlyAccount']

export type Notification = Notifications['schemas']['Notification']
export type Inbox = Notifications['schemas']['Inbox']

export type Analytics = Admin['schemas']['Analytics']
export type ExportList = Admin['schemas']['ExportList']
export type ExportItem = Admin['schemas']['Export']
