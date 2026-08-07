package exporthtml

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"time"

	"github.com/repolens/repolens/internal/model"
)

func Render(source *model.Analysis, privacy model.Privacy) ([]byte, error) {
	analysis := clone(source)
	applyPrivacy(analysis, privacy)
	payload, err := json.Marshal(analysis)
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := reportTemplate.Execute(&output, struct {
		Title   string
		Payload template.JS
		Year    int
	}{
		Title:   analysis.Repository.Name + " · RepoLens",
		Payload: template.JS(payload), // encoding/json escapes script-significant angle brackets.
		Year:    time.Now().Year(),
	}); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func clone(source *model.Analysis) *model.Analysis {
	data, _ := json.Marshal(source)
	var result model.Analysis
	_ = json.Unmarshal(data, &result)
	return &result
}

func applyPrivacy(analysis *model.Analysis, privacy model.Privacy) {
	analysis.ID = "offline-report"
	analysis.Config.Privacy = privacy
	if !privacy.ShowRepositoryName {
		analysis.Repository.Name = "私有仓库"
	}
	if !privacy.ShowCommitSHA {
		analysis.Repository.CommitSHA = ""
	}
	identityMap := map[string]string{}
	for index := range analysis.Contributors {
		contributor := &analysis.Contributors[index]
		if !privacy.ShowContributors {
			anonymousID := fmt.Sprintf("contributor-%02d", index+1)
			identityMap[contributor.ID] = anonymousID
			contributor.ID = anonymousID
			contributor.Name = fmt.Sprintf("贡献者 %02d", index+1)
			contributor.AvatarURL = ""
		} else if !privacy.ShowAvatars {
			contributor.AvatarURL = ""
		}
		if !privacy.ShowDirectories {
			contributor.TopDirectories = nil
		}
		for commitIndex := range contributor.RecentCommits {
			if !privacy.ShowCommitMessages {
				contributor.RecentCommits[commitIndex].Message = "提交记录"
			}
			if !privacy.ShowCommitSHA {
				contributor.RecentCommits[commitIndex].SHA = ""
			}
		}
	}
	if !privacy.ShowContributors {
		for index := range analysis.Timeline {
			if analysis.Timeline[index].Contributors == nil {
				continue
			}
			anonymous := make(map[string]model.Change, len(analysis.Timeline[index].Contributors))
			for id, change := range analysis.Timeline[index].Contributors {
				if replacement, ok := identityMap[id]; ok {
					anonymous[replacement] = change
				}
			}
			analysis.Timeline[index].Contributors = anonymous
		}
	}
}

var reportTemplate = template.Must(template.New("report").Parse(`<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="color-scheme" content="light">
<title>{{.Title}}</title>
<style>
:root{font-family:Inter,"Segoe UI","Microsoft YaHei",sans-serif;color:#171a1f;background:#f5f7f8;letter-spacing:0;--ink:#171a1f;--muted:#66717d;--line:#dfe4e8;--teal:#008f83;--coral:#e7664c;--yellow:#d7a320;--blue:#3d6fe0}*{box-sizing:border-box}body{margin:0}header{background:#fff;border-bottom:1px solid var(--line)}.head,.main{width:min(1180px,calc(100% - 40px));margin:auto}.head{height:70px;display:flex;align-items:center;justify-content:space-between}.brand{font-size:20px;font-weight:750}.brand b{color:var(--teal)}.meta{font-size:12px;color:var(--muted)}.main{padding:34px 0 64px}.title{display:flex;align-items:end;justify-content:space-between;gap:24px;margin-bottom:26px}h1{font-size:30px;margin:0 0 7px}.sub{color:var(--muted);font-size:14px}.status{border:1px solid #a9d8d3;color:#076c63;background:#eaf7f5;padding:6px 10px;border-radius:6px;font-size:12px;font-weight:650}.metrics{display:grid;grid-template-columns:repeat(4,1fr);border:1px solid var(--line);background:#fff;margin-bottom:24px}.metric{padding:20px;border-right:1px solid var(--line)}.metric:last-child{border:0}.metric label{display:block;color:var(--muted);font-size:12px;margin-bottom:8px}.metric strong{font-size:25px}.section{background:#fff;border:1px solid var(--line);margin-top:18px}.section-head{padding:17px 20px;border-bottom:1px solid var(--line);display:flex;justify-content:space-between;align-items:center}.section-head h2{font-size:15px;margin:0}.section-head span{font-size:12px;color:var(--muted)}table{border-collapse:collapse;width:100%}th,td{text-align:left;padding:14px 20px;border-bottom:1px solid #edf0f2;font-size:13px}th{font-size:11px;color:var(--muted);text-transform:uppercase;background:#fafbfb}.person{display:flex;align-items:center;gap:10px;font-weight:650}.avatar{width:30px;height:30px;border-radius:50%;display:grid;place-items:center;background:#d9efec;color:#076c63}.share{display:flex;align-items:center;gap:10px}.track{height:7px;width:110px;background:#edf0f1;border-radius:4px;overflow:hidden}.fill{height:100%;background:var(--teal)}.number{font-variant-numeric:tabular-nums}.categories{display:grid;grid-template-columns:repeat(3,1fr);gap:0}.category{padding:18px 20px;border-right:1px solid var(--line);border-bottom:1px solid var(--line)}.category:nth-child(3n){border-right:0}.category-name{font-size:12px;color:var(--muted)}.category strong{display:block;font-size:18px;margin-top:7px}.foot{color:var(--muted);font-size:11px;line-height:1.7;margin-top:18px}.empty{padding:28px;color:var(--muted)}@media(max-width:760px){.head,.main{width:min(100% - 24px,1180px)}.meta{display:none}.title{align-items:start;flex-direction:column}.metrics{grid-template-columns:1fr 1fr}.metric:nth-child(2){border-right:0}.metric:nth-child(-n+2){border-bottom:1px solid var(--line)}.section{overflow:auto}.categories{grid-template-columns:1fr 1fr}.category:nth-child(3n){border-right:1px solid var(--line)}.category:nth-child(2n){border-right:0}.track{width:70px}th,td{padding:12px;white-space:nowrap}}
</style>
</head>
<body>
<header><div class="head"><div class="brand">Repo<b>Lens</b></div><div class="meta" id="generated"></div></div></header>
<main class="main"><div class="title"><div><h1 id="repo"></h1><div class="sub" id="scope"></div></div><div class="status" id="mode"></div></div><div class="metrics" id="metrics"></div><section class="section"><div class="section-head"><h2>贡献者对比</h2><span>综合占比及独立维度</span></div><div id="contributors"></div></section><section class="section"><div class="section-head"><h2>代码类型</h2><span>有效新增与删除</span></div><div class="categories" id="categories"></div></section><p class="foot" id="method"></p></main>
<script>const a={{.Payload}};const n=v=>new Intl.NumberFormat('zh-CN').format(v||0);const p=v=>(v||0).toFixed(1)+'%';document.getElementById('repo').textContent=a.repository.name;document.getElementById('scope').textContent=a.repository.ref+(a.repository.commitSha?' · '+a.repository.commitSha.slice(0,10):'')+' · '+n(a.summary.commits)+' 次提交';document.getElementById('generated').textContent='生成于 '+new Date(a.generatedAt).toLocaleString('zh-CN');document.getElementById('mode').textContent=a.mode==='git-only'?'本地 Git 分析':'GitHub 增强分析';const metrics=[['贡献者',n(a.summary.contributors)],['有效新增','+'+n(a.summary.additions)],['有效删除','-'+n(a.summary.deletions)],['归属覆盖率',p(a.summary.coveragePercent)]];document.getElementById('metrics').innerHTML=metrics.map(x=>'<div class="metric"><label>'+x[0]+'</label><strong>'+x[1]+'</strong></div>').join('');const rows=a.contributors.map((c,i)=>'<tr><td><div class="person"><span class="avatar">'+(i+1)+'</span>'+escapeHTML(c.name)+'</div></td><td><div class="share"><span class="track"><span class="fill" style="width:'+Math.min(c.compositeShare,100)+'%"></span></span><b class="number">'+p(c.compositeShare)+'</b></div></td><td class="number">'+p(c.workloadShare)+'</td><td class="number">'+p(c.ownershipShare)+'</td><td class="number">'+p(c.retentionRate)+'</td><td class="number">'+n(c.commits)+'</td></tr>').join('');document.getElementById('contributors').innerHTML=rows?'<table><thead><tr><th>贡献者</th><th>综合占比</th><th>工作量</th><th>当前归属</th><th>留存率</th><th>提交</th></tr></thead><tbody>'+rows+'</tbody></table>':'<div class="empty">没有可显示的贡献者</div>';const labels={source:'源代码',test:'测试',docs:'文档',config:'配置',asset:'资源',other:'其他'};document.getElementById('categories').innerHTML=a.categories.map(c=>'<div class="category"><span class="category-name">'+labels[c.category]+'</span><strong>+'+n(c.additions)+' / -'+n(c.deletions)+'</strong></div>').join('');document.getElementById('method').textContent='RepoLens '+a.schemaVersion+' · 配置指纹 '+a.config.fingerprint+' · 本报告不包含源代码片段、文件名、邮箱、本机路径或凭据。综合占比是可配置的统计指标，不代表对个人价值或代码质量的判断。';function escapeHTML(v){const e=document.createElement('span');e.textContent=v;return e.innerHTML}</script>
</body></html>`))
