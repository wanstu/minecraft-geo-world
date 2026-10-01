const $=id=>document.getElementById(id);
let active=null, layer='final', poll=null;

async function api(url, opts={}){
  const r=await fetch(url,{headers:{'Content-Type':'application/json'},...opts});
  const data=await r.json().catch(()=>({}));
  if(!r.ok) throw new Error(data.error||('HTTP '+r.status));
  return data;
}

function setBusy(v){
  $('generate').disabled=v;
  $('save').disabled=v;
  $('load').disabled=v;
}

async function refreshStatus(){
  try{
    const s=await api('/api/status');
    if(s.generating){
      const pct=s.total?Math.round(s.completed/s.total*100):0;
      $('progressBar').style.width=pct+'%';
      $('progressText').textContent=`正在生成：${s.stage||'准备'} · ${s.completed}/${s.total} · ${(s.elapsedMs/1000).toFixed(1)}s`;
      $('topStatus').textContent='生成中';
    }
  }catch(e){}
}

$('generate').onclick=async()=>{
  setBusy(true);
  $('progressBar').style.width='0%';
  $('progressText').textContent='开始生成…';
  poll=setInterval(refreshStatus,250);
  try{
    active=await api('/api/generate',{method:'POST',body:JSON.stringify({
      seed:Number($('seed').value),width:Number($('width').value),height:Number($('height').value),
      cellSize:Number($('cellSize').value),plateCount:Number($('plates').value),oceanFraction:Number($('ocean').value)
    })});
    $('progressBar').style.width='100%';
    $('progressText').textContent=`生成完成 · ${(active.durationMs/1000).toFixed(2)}s`;
    $('topStatus').textContent='已生成';
    renderStats();
    loadMap();
    await refreshModels();
  }catch(e){
    $('progressText').innerHTML='<span class="error">'+escapeHtml(e.message)+'</span>';
    $('topStatus').textContent='生成失败';
  }finally{
    clearInterval(poll);poll=null;setBusy(false);
  }
};

document.querySelectorAll('.layer').forEach(btn=>btn.onclick=()=>{
  document.querySelectorAll('.layer').forEach(b=>b.classList.remove('active'));
  btn.classList.add('active'); layer=btn.dataset.layer; if(active)loadMap();
});

function loadMap(){
  $('mapWrap').classList.remove('empty');
  $('map').src=`/api/layer?name=${encodeURIComponent(layer)}&width=720&height=450&t=${Date.now()}`;
}

$('map').onclick=async e=>{
  if(!active)return;
  const rect=$('map').getBoundingClientRect();
  const fx=(e.clientX-rect.left)/rect.width, fz=(e.clientY-rect.top)/rect.height;
  const g=active.geometry;
  const x=g.OriginX+fx*g.SpanXBlocks, z=g.OriginZ+fz*g.SpanZBlocks;
  const c=$('cursor'); c.style.display='block';
  c.style.left=(e.clientX-$('mapWrap').getBoundingClientRect().left-6)+'px';
  c.style.top=(e.clientY-$('mapWrap').getBoundingClientRect().top-6)+'px';
  try{ renderSample(await api(`/api/sample?x=${x}&z=${z}`)); }catch(err){alert(err.message)}
};

$('save').onclick=async()=>{
  try{
    const r=await api('/api/save',{method:'POST',body:JSON.stringify({name:$('modelName').value})});
    $('modelMessage').innerHTML='<span class="ok">已保存：'+escapeHtml(r.name)+'.geoworld</span>';
    await refreshModels();
  }catch(e){$('modelMessage').innerHTML='<span class="error">'+escapeHtml(e.message)+'</span>'}
};

$('load').onclick=async()=>{
  const name=$('models').value;if(!name)return;
  setBusy(true);
  try{
    active=await api('/api/load',{method:'POST',body:JSON.stringify({name})});
    $('seed').value=active.seed;
    $('width').value=active.geometry.Width;$('height').value=active.geometry.Height;
    $('topStatus').textContent='已重开';
    $('progressText').textContent=`模型重开完成 · ${active.durationMs}ms`;
    renderStats();loadMap();
  }catch(e){$('modelMessage').innerHTML='<span class="error">'+escapeHtml(e.message)+'</span>'}
  finally{setBusy(false)}
};

async function refreshModels(){
  try{
    const list=await api('/api/models');
    $('models').innerHTML=list.length?list.map(m=>`<option value="${escapeAttr(m.name)}">${escapeHtml(m.name)} · Seed ${m.seed} · ${formatBytes(m.fileBytes)}</option>`).join(''):'<option value="">暂无</option>';
  }catch(e){}
}

function renderStats(){
  if(!active)return;
  const g=active.geometry;
  $('stats').outerHTML=`<div id="stats">
    ${kv('来源',active.source==='generated'?'刚生成':'模型重开')}
    ${kv('Seed',active.seed)}
    ${kv('宏观网格',g.Width+' × '+g.Height)}
    ${kv('世界范围',Math.round(g.SpanXBlocks)+' × '+Math.round(g.SpanZBlocks)+' blocks')}
    ${kv('河段',active.riverSegments)}
    ${kv('湖泊节点',active.lakeNodes)}
  </div>`;
}

function renderSample(s){
  $('sampleEmpty').hidden=true;$('sample').hidden=false;
  const water=s.waterSurfaceDefined?`${s.waterKind} · 水面 ${f(s.waterSurfaceElevation)} · 深度 ${f(s.waterDepth)}`:s.waterKind;
  $('sample').innerHTML=`
    <div class="sample-section"><h3>最终 Runtime</h3>
      ${kv('坐标',Math.round(s.x)+', '+Math.round(s.z))}
      ${kv('Ground',f(s.groundHeight))}
      ${kv('水体',water)}
      ${kv('Ocean eligibility',f(s.oceanEligibility))}
      ${kv('River weight',f(s.riverWeight))}
    </div>
    <div class="sample-section"><h3>生态 / Minecraft Profile</h3>
      ${kv('生态区',s.ecoRegion)}
      ${kv('Biome',s.biomeFamily)}
      ${kv('Terrain',s.terrainVariant)}
      ${kv('Surface',s.surfaceProfile)}
      ${kv('Vegetation',s.vegetationProfile)}
      ${kv('Profile confidence',f(s.profileConfidence))}
    </div>
    <div class="sample-section"><h3>一致性</h3>
      ${kv('来源节点',s.sourceNode)}
      ${kv('来源陆地',s.sourceLand?'是':'否')}
      ${kv('Semantic fallback',s.semanticFallback?'<span class="error">是</span>':'<span class="ok">否</span>')}
      ${kv('Semantic mismatch',s.semanticMismatch?'<span class="error">是</span>':'<span class="ok">否</span>')}
    </div>`;
}

function kv(k,v){return `<div class="kv"><span>${k}</span><b>${v}</b></div>`}
function f(v){return Number(v).toFixed(6)}
function formatBytes(n){return n<1048576?(n/1024).toFixed(1)+' KB':(n/1048576).toFixed(2)+' MB'}
function escapeHtml(s){return String(s).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]))}
function escapeAttr(s){return escapeHtml(s)}
refreshModels();
