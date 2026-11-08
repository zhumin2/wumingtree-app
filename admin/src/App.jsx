import { useEffect, useState } from 'react'
import {
  Alert, Button, Form, Input, Layout, message, Modal, Select, Space, Table, Tag, Typography,
} from 'antd'
import { api, getToken, setToken } from './api.js'

const { Header, Content } = Layout
const { Title, Text } = Typography

const STATUS = {
  submitted: { label: '待审核', color: 'gold' },
  approved: { label: '已通过', color: 'green' },
  rejected: { label: '已拒绝', color: 'red' },
}

function fmtTime(v) {
  if (!v) return '-'
  return new Date(v).toLocaleString('zh-CN', { hour12: false })
}

function LoginPage({ onLogin }) {
  const [form] = Form.useForm()
  const [countdown, setCountdown] = useState(0)
  const [sending, setSending] = useState(false)
  const [logging, setLogging] = useState(false)

  useEffect(() => {
    if (countdown <= 0) return undefined
    const t = setInterval(() => setCountdown((c) => c - 1), 1000)
    return () => clearInterval(t)
  }, [countdown])

  const sendCode = async () => {
    const phone = form.getFieldValue('phone')
    if (!/^1[3-9]\d{9}$/.test(phone || '')) {
      message.warning('请填写正确的手机号')
      return
    }
    setSending(true)
    try {
      await api('/auth/sms-codes', { method: 'POST', body: { phone } })
      setCountdown(60)
      message.success('验证码已发送')
    } catch (e) {
      message.error(e.message)
    } finally {
      setSending(false)
    }
  }

  const login = async (v) => {
    setLogging(true)
    try {
      const res = await api('/auth/login', { method: 'POST', body: { phone: v.phone, code: v.code } })
      if (res.outcome !== 'member') {
        message.error('该账号不是管理员，或尚未通过审核')
        return
      }
      if (res.member.role !== 'admin') {
        message.error('该账号不是管理员')
        return
      }
      setToken(res.tokens.accessToken)
      onLogin()
    } catch (e) {
      message.error(e.message)
    } finally {
      setLogging(false)
    }
  }

  return (
    <div style={{ maxWidth: 380, margin: '96px auto' }}>
      <Title level={3} style={{ textAlign: 'center' }}>无名树 · 管理台</Title>
      <Text type="secondary" style={{ display: 'block', textAlign: 'center', marginBottom: 24 }}>
        仅限管理员账号（短信验证码登录）
      </Text>
      <Form form={form} onFinish={login}>
        <Form.Item name="phone" rules={[
          { required: true, message: '请填写手机号' },
          { pattern: /^1[3-9]\d{9}$/, message: '手机号格式不正确' },
        ]}>
          <Input placeholder="手机号" />
        </Form.Item>
        <Form.Item name="code" rules={[{ required: true, message: '请填写验证码' }]}>
          <Space.Compact style={{ width: '100%' }}>
            <Input placeholder="6 位验证码" maxLength={6} />
            <Button disabled={sending || countdown > 0} onClick={sendCode}>
              {countdown > 0 ? `${countdown}s` : '发送验证码'}
            </Button>
          </Space.Compact>
        </Form.Item>
        <Button type="primary" htmlType="submit" block loading={logging}>登录</Button>
      </Form>
    </div>
  )
}

function ApplicationsPage({ onLogout }) {
  const [rows, setRows] = useState([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [status, setStatus] = useState('submitted')
  const [loading, setLoading] = useState(false)
  const [me, setMe] = useState(null)
  const [rejecting, setRejecting] = useState(null) // 待拒绝的申请
  const [reasons, setReasons] = useState([])
  const [reason, setReason] = useState(null)
  const pageSize = 10

  const load = async (p = page, s = status) => {
    setLoading(true)
    try {
      const data = await api(`/admin/applications?status=${s}&offset=${(p - 1) * pageSize}&limit=${pageSize}`)
      setRows(data.items)
      setTotal(data.total)
    } catch (e) {
      message.error(e.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    api('/me').then(setMe).catch((e) => message.error(e.message))
  }, [])

  useEffect(() => { load(page, status) }, [page, status])

  const approve = async (row) => {
    try {
      await api(`/admin/applications/${row.id}/approve`, { method: 'POST' })
      message.success(`已通过「${row.nickname}」的申请，成员已创建`)
      load()
    } catch (e) {
      message.error(e.message)
    }
  }

  const openReject = async (row) => {
    setRejecting(row)
    try {
      setReasons(await api('/admin/reject-reasons'))
    } catch (e) {
      message.error(e.message)
    }
  }

  const doReject = async () => {
    if (!reason) {
      message.warning('请选择拒绝原因')
      return
    }
    try {
      await api(`/admin/applications/${rejecting.id}/reject`, { method: 'POST', body: { reason } })
      message.success('已拒绝并通知模板原因')
      setRejecting(null)
      setReason(null)
      load()
    } catch (e) {
      message.error(e.message)
    }
  }

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    { title: '称呼', dataIndex: 'nickname', width: 120 },
    {
      title: '状态', dataIndex: 'status', width: 100,
      render: (s) => <Tag color={STATUS[s]?.color}>{STATUS[s]?.label || s}</Tag>,
    },
    { title: '提交时间', dataIndex: 'createdAt', width: 170, render: fmtTime },
    { title: '审核时间', dataIndex: 'reviewedAt', width: 170, render: fmtTime },
    { title: '拒绝原因', dataIndex: 'rejectReason', ellipsis: true },
    {
      title: '操作', width: 170,
      render: (_, row) => row.status === 'submitted' ? (
        <Space>
          <Button size="small" type="primary" onClick={() => approve(row)}>通过</Button>
          <Button size="small" danger onClick={() => openReject(row)}>拒绝</Button>
        </Space>
      ) : <Text type="secondary">已处理（{row.reviewedBy ? `管理员 #${row.reviewedBy}` : '-'}）</Text>,
    },
  ]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{ background: '#fff', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={4} style={{ margin: 0 }}>无名树 · 管理台</Title>
        <Space>
          <Text type="secondary">{me ? `${me.nickname}（管理员）` : ''}</Text>
          <Button onClick={async () => { setToken(null); onLogout() }}>退出</Button>
        </Space>
      </Header>
      <Content style={{ padding: 24 }}>
        <Space style={{ marginBottom: 16 }}>
          <Select
            value={status}
            style={{ width: 140 }}
            onChange={(s) => { setPage(1); setStatus(s) }}
            options={[
              { value: 'submitted', label: '待审核' },
              { value: 'approved', label: '已通过' },
              { value: 'rejected', label: '已拒绝' },
              { value: '', label: '全部' },
            ]}
          />
          <Button onClick={() => load()}>刷新</Button>
        </Space>
        <Table
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={rows}
          pagination={{ current: page, pageSize, total, onChange: setPage }}
        />
        <Modal
          open={!!rejecting}
          title={`拒绝「${rejecting?.nickname}」的申请`}
          onOk={doReject}
          onCancel={() => setRejecting(null)}
          okText="确认拒绝"
          okButtonProps={{ danger: true }}
        >
          <Alert
            style={{ marginBottom: 12 }}
            type="info"
            showIcon
            message="拒绝后可重新申请；原因会以私信通知申请者。"
          />
          <Select
            style={{ width: '100%' }}
            placeholder="选择原因模板"
            value={reason}
            onChange={setReason}
            options={reasons.map((r) => ({ value: r, label: r }))}
          />
        </Modal>
      </Content>
    </Layout>
  )
}

export default function App() {
  const [authed, setAuthed] = useState(!!getToken())
  return authed
    ? <ApplicationsPage onLogout={() => setAuthed(false)} />
    : <LoginPage onLogin={() => setAuthed(true)} />
}
