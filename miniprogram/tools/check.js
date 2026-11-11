// 结构校验：app.json pages 与文件一一对应；全部 JSON 可解析；红线静态检查。
// CI 与本地均可运行：node tools/check.js
const fs = require('fs');
const path = require('path');

const root = path.join(__dirname, '..');
const errors = [];

// 1. app.json 可解析且 pages 存在
let app;
try {
  app = JSON.parse(fs.readFileSync(path.join(root, 'app.json'), 'utf8'));
} catch (e) {
  console.error('app.json 解析失败: ' + e.message);
  process.exit(1);
}
if (!Array.isArray(app.pages) || app.pages.length === 0) {
  console.error('app.json pages 为空');
  process.exit(1);
}
for (const p of app.pages) {
  for (const ext of ['.js', '.json', '.wxml', '.wxss']) {
    if (!fs.existsSync(path.join(root, p + ext))) {
      errors.push('缺少页面文件: ' + p + ext);
    }
  }
}

// 2. 全部 JSON 可解析
function walk(dir) {
  for (const name of fs.readdirSync(dir)) {
    const f = path.join(dir, name);
    const st = fs.statSync(f);
    if (st.isDirectory()) {
      if (name !== 'node_modules') walk(f);
      continue;
    }
    if (name.endsWith('.json')) {
      try {
        JSON.parse(fs.readFileSync(f, 'utf8'));
      } catch (e) {
        errors.push('JSON 解析失败: ' + f + ' — ' + e.message);
      }
    }
  }
}
walk(root);

// 3. 红线静态检查：sitemap 禁止索引（不做公开分发）
const sitemap = JSON.parse(fs.readFileSync(path.join(root, 'sitemap.json'), 'utf8'));
const disallowAll = (sitemap.rules || []).some((r) => r.action === 'disallow' && r.page === '*');
if (!disallowAll) {
  errors.push('红线：sitemap 必须对全部页面 disallow（不做公开分发）');
}

// 4. 红线静态检查： crises 求助卡文案必须存在于详情页（危机分支）
const detailWxml = fs.readFileSync(path.join(root, 'pages/hollow/detail.wxml'), 'utf8');
if (detailWxml.indexOf('crisis') === -1) {
  errors.push('红线：帖子详情缺少 crisis 求助卡渲染');
}

// 5. 首页常驻风险提示
const homeWxml = fs.readFileSync(path.join(root, 'pages/home/home.wxml'), 'utf8');
if (homeWxml.indexOf('同伴互助不能替代专业医疗或心理支持') === -1) {
  errors.push('红线：首页缺少常驻风险提示');
}

// 6. 上线必查项：协议三件套已注册，gate 有勾选与链接，我的页有入口
const agreeWxml = fs.readFileSync(path.join(root, 'pages/agreement/agreement.wxml'), 'utf8');
if (agreeWxml.indexOf('sections') === -1) {
  errors.push('协议页缺少正文渲染');
}
const gateWxml = fs.readFileSync(path.join(root, 'pages/gate/gate.wxml'), 'utf8');
for (const doc of ['user', 'privacy', 'rules']) {
  if (gateWxml.indexOf('data-type="' + doc + '"') === -1) {
    errors.push('红线：邀请页缺少《' + doc + '》协议链接');
  }
}
if (gateWxml.indexOf('toggleAgree') === -1) {
  errors.push('红线：邀请页缺少协议勾选');
}
const meWxml = fs.readFileSync(path.join(root, 'pages/me/me.wxml'), 'utf8');
if (meWxml.indexOf('openAgreement') === -1) {
  errors.push('红线：我的页缺少协议入口');
}
const agreeContent = require(path.join(root, 'pages/agreement/content.js'));
for (const key of ['user', 'privacy', 'rules']) {
  if (!agreeContent.DOCS[key] || !agreeContent.DOCS[key].sections.length) {
    errors.push('协议正文缺失: ' + key);
  }
}

if (errors.length) {
  console.error(errors.join('\n'));
  process.exit(1);
}
console.log('miniprogram structure OK: ' + app.pages.length + ' pages');
