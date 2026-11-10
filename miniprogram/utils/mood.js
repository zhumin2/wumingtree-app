// 心情枚举（服务端 share/tired/ask/talk；颜色类在 app.wxss mood-*）。
const MOODS = [
  { code: 'share', label: '想分享' },
  { code: 'tired', label: '有点累' },
  { code: 'ask', label: '求经验' },
  { code: 'talk', label: '只是想说说' },
];

function moodLabel(code) {
  for (let i = 0; i < MOODS.length; i++) {
    if (MOODS[i].code === code) return MOODS[i].label;
  }
  return '想分享';
}

module.exports = {
  MOODS: MOODS,
  moodLabel: moodLabel,
};
