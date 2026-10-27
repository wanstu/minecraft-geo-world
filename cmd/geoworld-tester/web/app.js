const $=id=>document.getElementById(id);

let active=null;
let layer='final';
let poll=null;
let busy=false;
let serviceOnline=true;
let mapLoadToken=0;
let mapObjectURL=null;
let renderedSeed=null;
let modelNameTouched=false;
let exportNameTouched=false;
let lastExport=null;
const MAX_MAP_ZOOM=64;
const FULL_DETAIL_ZOOM=MAX_MAP_ZOOM;
let mapZoom=1;
let mapPanX=0;
let mapPanY=0;
let mapDragging=false;
let mapDragMoved=false;
let mapDragStartX=0;
let mapDragStartY=0;
let mapPanStartX=0;
let mapPanStartY=0;
let suppressMapClick=false;
let lastFocusPoint=null;
let previewFeatures=null;
let mineralBodies=null;
let mineralLoadToken=0;
let mineralRefreshTimer=null;
let mineralBridgeUnavailable=false;
let structureCandidates=null;
let structureLoadToken=0;
let structureRefreshTimer=null;
let touchPoints=new Map();
let pinchStartDistance=0;
let pinchStartZoom=1;
let acceptanceData=null;
let currentPage='world';

const stageNames={
  phase1:'地壳与大尺度地形',
  climate:'气候',
  hydrology:'水系',
  erosion:'侵蚀',
  landform:'地貌',
  geomorph:'地形结构',
  ecology:'生态',
  mcprofile:'Minecraft 配置',
  terrain:'连续地形',
  reconcile:'海陆校正',
  waterbed:'河床与湖床',
  worldsample:'最终地面与水面',
  runtimeprofile:'最终群系配置'
};


const continentPresets={
  default:{
    ocean:.68,fragmentation:.35,separation:.65,
    help:'64 Seed 基线（Seed 1–64，320×200 / 128）：平均主要陆块 2.61、全部陆块 6.14、最大陆块占比 69.9%。'
  },
  super:{
    ocean:.50,fragmentation:.05,separation:.05,
    help:'64 Seed 基线：50/64 达到“主要陆块=1 且最大陆块≥85%”；平均主要陆块 1.23、最大陆块占比 96.8%。'
  },
  multi:{
    ocean:.68,fragmentation:.30,separation:.90,
    help:'64 Seed 基线：54/64 达到“主要陆块≥2 且最大陆块≤80%”；平均主要陆块 3.48、最大陆块占比 57.0%。'
  },
  broken:{
    ocean:.68,fragmentation:.75,separation:.75,
    help:'64 Seed 基线：平均主要陆块 2.69、全部陆块 8.86；强调海峡、半岛和次级陆块，不保证没有大型大陆。'
  },
  archipelago:{
    ocean:.90,fragmentation:.95,separation:.95,
    help:'64 Seed 基线：62/64 达到“全部陆块≥10 且最大陆块≤65%”；平均主要陆块 6.73、全部陆块 15.61、最大陆块占比 33.4%。'
  }
};

class ApiError extends Error{
  constructor(message,{network=false,status=0}={}){
    super(message);
    this.network=network;
    this.status=status;
  }
}

function backendMessage(){
  return '无法连接 GeoWorld Tester 后端。请确认启动 Tester 的命令窗口仍在运行；当前地图没有更新。';
}

async function api(url,opts={}){
  let r;
  try{
    r=await fetch(url,{headers:{'Content-Type':'application/json'},...opts});
  }catch(err){
    updateBackend(false);
    throw new ApiError(backendMessage(),{network:true});
  }
  updateBackend(true);
  const data=await r.json().catch(()=>({}));
  if(!r.ok)throw new ApiError(data.error||('HTTP '+r.status),{status:r.status});
  return data;
}

function updateBackend(online){
  const wasOnline=serviceOnline;
  serviceOnline=online;
  const el=$('backendStatus');
  el.textContent=online?'后端在线':'后端断开';
  el.classList.toggle('online',online);
  el.classList.toggle('offline',!online);

  if(!online){
    $('topStatus').textContent='服务断开';
    if(active||renderedSeed!==null){
      setMapNotice('后端已断开；当前仍显示上一次成功加载的地图。','error');
    }
  }else if(!wasOnline){
    const staleSeed=renderedSeed;
    active=null;
    previewFeatures=null;
    resetMineralPreview();
    resetStructurePreview();
    clearSample();
    clearExportResult();
    clearAcceptanceView();
    renderFeatureOverlay();
    renderHomeCurrentWorld();
    renderEmptyStats();
    $('topStatus').textContent='需重新加载';
    $('progressText').textContent='后端已恢复，请重新生成世界或重新打开已保存模型';
    if(staleSeed!==null){
      setMapNotice(`后端已恢复；当前仍是 Seed ${staleSeed} 的旧图，请重新生成或重新打开模型。`,'warn');
    }else{
      setMapNotice('');
    }
    void refreshModels();
  }
  updateActionState();
}

function setBusy(v){
  busy=v;
  updateActionState();
}

function clearExportResult({message=true}={}){
  lastExport=null;
  $('openExport').hidden=true;
  $('exportBounds').hidden=true;
  $('spawnMarker').hidden=true;
  if(message)$('exportMessage').textContent='生成或重开模型后即可导出。';
  updateActionState();
}

function clampMapZoom(v){
  return Math.max(1,Math.min(MAX_MAP_ZOOM,v));
}

function applyMapTransform(){
  const img=$('map');
  const wrap=$('mapWrap');
  img.style.transform=`translate(${mapPanX}px,${mapPanY}px) scale(${mapZoom})`;
  wrap.classList.toggle('zoomed',mapZoom>1.001);
  $('mapZoomLabel').textContent=mapZoom.toFixed(1)+'×';
  updateActionState();
  requestAnimationFrame(()=>{
    renderExportOverlay();
    renderCursorOverlay();
    positionFeatureOverlay();
    renderFeatureOverlay();
    scheduleMineralRefresh();
    scheduleStructureRefresh();
  });
}

function resetMapView(){
  mapZoom=1;
  mapPanX=0;
  mapPanY=0;
  applyMapTransform();
}

function zoomMapAt(clientX,clientY,nextZoom){
  nextZoom=clampMapZoom(nextZoom);
  if(Math.abs(nextZoom-mapZoom)<1e-6)return;
  if(nextZoom<=1.001){
    resetMapView();
    return;
  }
  const img=$('map');
  const oldRect=img.getBoundingClientRect();
  if(oldRect.width<=0||oldRect.height<=0)return;
  const fx=(clientX-oldRect.left)/oldRect.width;
  const fz=(clientY-oldRect.top)/oldRect.height;

  mapZoom=nextZoom;
  img.style.transform=`translate(${mapPanX}px,${mapPanY}px) scale(${mapZoom})`;
  const nextRect=img.getBoundingClientRect();
  mapPanX+=clientX-(nextRect.left+fx*nextRect.width);
  mapPanY+=clientY-(nextRect.top+fz*nextRect.height);
  applyMapTransform();
}

function zoomMapCentered(factor){
  const wrap=$('mapWrap').getBoundingClientRect();
  zoomMapAt(
    wrap.left+wrap.width/2,
    wrap.top+wrap.height/2,
    mapZoom*factor
  );
}

function renderCursorOverlay(){
  const c=$('cursor');
  if(!active||!lastFocusPoint||$('mapWrap').classList.contains('empty')){
    c.style.display='none';
    return;
  }
  const g=active.geometry;
  const fx=Math.max(0,Math.min(1,(lastFocusPoint.x-g.OriginX)/g.SpanXBlocks));
  const fz=Math.max(0,Math.min(1,(lastFocusPoint.z-g.OriginZ)/g.SpanZBlocks));
  const imgRect=$('map').getBoundingClientRect();
  const wrapRect=$('mapWrap').getBoundingClientRect();
  if(imgRect.width<=0||imgRect.height<=0){
    c.style.display='none';
    return;
  }
  c.style.display='block';
  c.style.left=(imgRect.left-wrapRect.left+fx*imgRect.width-6)+'px';
  c.style.top=(imgRect.top-wrapRect.top+fz*imgRect.height-6)+'px';
}

function renderExportOverlay(){
  const bounds=$('exportBounds');
  const spawn=$('spawnMarker');
  if(!active||!lastExport||$('mapWrap').classList.contains('empty')){
    bounds.hidden=true;
    spawn.hidden=true;
    return;
  }

  const g=active.geometry;
  const worldMinX=(lastExport.centerChunkX-lastExport.radiusChunks)*16;
  const worldMaxX=(lastExport.centerChunkX+lastExport.radiusChunks+1)*16;
  const worldMinZ=(lastExport.centerChunkZ-lastExport.radiusChunks)*16;
  const worldMaxZ=(lastExport.centerChunkZ+lastExport.radiusChunks+1)*16;
  const modelMinX=g.OriginX;
  const modelMaxX=g.OriginX+g.SpanXBlocks;
  const modelMinZ=g.OriginZ;
  const modelMaxZ=g.OriginZ+g.SpanZBlocks;

  if(
    worldMaxX<=modelMinX||worldMinX>=modelMaxX||
    worldMaxZ<=modelMinZ||worldMinZ>=modelMaxZ
  ){
    bounds.hidden=true;
    spawn.hidden=true;
    return;
  }

  const fx0=Math.max(0,Math.min(1,(worldMinX-modelMinX)/g.SpanXBlocks));
  const fx1=Math.max(0,Math.min(1,(worldMaxX-modelMinX)/g.SpanXBlocks));
  const fz0=Math.max(0,Math.min(1,(worldMinZ-modelMinZ)/g.SpanZBlocks));
  const fz1=Math.max(0,Math.min(1,(worldMaxZ-modelMinZ)/g.SpanZBlocks));

  const imgRect=$('map').getBoundingClientRect();
  const wrapRect=$('mapWrap').getBoundingClientRect();
  if(imgRect.width<=0||imgRect.height<=0){
    bounds.hidden=true;
    spawn.hidden=true;
    return;
  }

  const baseLeft=imgRect.left-wrapRect.left;
  const baseTop=imgRect.top-wrapRect.top;
  bounds.hidden=false;
  bounds.style.left=(baseLeft+fx0*imgRect.width)+'px';
  bounds.style.top=(baseTop+fz0*imgRect.height)+'px';
  bounds.style.width=Math.max(2,(fx1-fx0)*imgRect.width)+'px';
  bounds.style.height=Math.max(2,(fz1-fz0)*imgRect.height)+'px';
  bounds.title='Minecraft 预生成范围：radius '+lastExport.radiusChunks+' Chunk';

  const sfx=Math.max(0,Math.min(1,(lastExport.spawnX-modelMinX)/g.SpanXBlocks));
  const sfz=Math.max(0,Math.min(1,(lastExport.spawnZ-modelMinZ)/g.SpanZBlocks));
  spawn.hidden=false;
  spawn.style.left=(baseLeft+sfx*imgRect.width)+'px';
  spawn.style.top=(baseTop+sfz*imgRect.height)+'px';
  spawn.title='Minecraft 出生点：'+lastExport.spawnX+', '+lastExport.spawnY+', '+lastExport.spawnZ;
}

function updateActionState(){
  $('generate').disabled=busy||!serviceOnline;
  $('save').disabled=busy||!serviceOnline||!active||active.source!=='generated';
  $('load').disabled=busy||!serviceOnline||!$('models').value;
  $('exportMinecraft').disabled=busy||!serviceOnline||!active;
  $('openExport').disabled=busy||!serviceOnline||!lastExport;
  const mapDisabled=busy||!serviceOnline||!active||$('mapWrap').classList.contains('empty');
  $('mapZoomIn').disabled=mapDisabled||mapZoom>=MAX_MAP_ZOOM-.001;
  $('mapZoomOut').disabled=mapDisabled||mapZoom<=1.001;
  $('mapZoomReset').disabled=mapDisabled||(mapZoom<=1.001&&Math.abs(mapPanX)<0.5&&Math.abs(mapPanY)<0.5);
  $('mobileZoomIn').disabled=mapDisabled||mapZoom>=MAX_MAP_ZOOM-.001;
  $('mobileZoomOut').disabled=mapDisabled||mapZoom<=1.001;
  $('jumpToCoord').disabled=mapDisabled;
  $('acceptanceGenerate').disabled=busy||!serviceOnline||!active;
  document.querySelectorAll('.acceptance-download').forEach(btn=>{btn.disabled=busy||!serviceOnline||!active});
}

function setMapNotice(text='',kind=''){
  const el=$('mapNotice');
  if(!text){
    el.hidden=true;
    el.textContent='';
    el.className='map-notice';
    return;
  }
  el.hidden=false;
  el.textContent=text;
  el.className='map-notice '+kind;
}

function stageLabel(stage){
  return stageNames[stage]||stage||'准备';
}

async function refreshStatus(){
  try{
    const s=await api('/api/status');
    if(s.generating){
      const pct=s.total?Math.round(s.completed/s.total*100):0;
      $('progressBar').style.width=pct+'%';
      $('progressText').textContent=`正在生成：${stageLabel(s.stage)} · ${s.completed}/${s.total} · ${((s.elapsedMs||0)/1000).toFixed(1)}s`;
      $('topStatus').textContent='生成中';
    }else if(s.exporting){
      $('topStatus').textContent='导出中';
    }
  }catch(e){
    if(e.network){
      $('topStatus').textContent='服务断开';
      if(active)setMapNotice('后端已断开；当前仍显示上一次成功加载的地图。','error');
    }
  }
}

function updateRangePreview(){
  const w=Math.max(1,Number($('width').value)||0);
  const h=Math.max(1,Number($('height').value)||0);
  const cell=Number($('cellSize').value)||0;
  const sx=Math.max(0,w-1)*cell;
  const sz=Math.max(0,h-1)*cell;
  $('rangePreview').textContent=`当前覆盖约 ${Math.round(sx).toLocaleString()} × ${Math.round(sz).toLocaleString()} 方块；间距越小，细节越密，但同样网格覆盖范围越小。`;
}


function nearlyEqual(a,b){
  return Math.abs(Number(a)-Number(b))<1e-9;
}

function matchingContinentPreset(){
  const ocean=Number($('ocean').value);
  const fragmentation=Number($('fragmentation').value);
  const separation=Number($('separation').value);
  for(const [name,p] of Object.entries(continentPresets)){
    if(nearlyEqual(ocean,p.ocean)&&
       nearlyEqual(fragmentation,p.fragmentation)&&
       nearlyEqual(separation,p.separation)){
      return name;
    }
  }
  return 'custom';
}

function updatePresetHelp(){
  const name=$('continentPreset').value;
  const p=continentPresets[name];
  if(!p){
    $('presetHelp').textContent='自定义：当前三个参数不属于内置预设；生成与保存仍直接使用输入框中的实际数值。';
    return;
  }
  $('presetHelp').textContent=
    `洋壳 ${p.ocean.toFixed(2)} · 破碎 ${p.fragmentation.toFixed(2)} · 分离 ${p.separation.toFixed(2)}。 ${p.help}`;
}

function syncContinentPreset(){
  $('continentPreset').value=matchingContinentPreset();
  updatePresetHelp();
}

function applyContinentPreset(name){
  const p=continentPresets[name];
  if(!p){
    updatePresetHelp();
    return;
  }
  $('ocean').value=p.ocean.toFixed(2);
  $('fragmentation').value=p.fragmentation.toFixed(2);
  $('separation').value=p.separation.toFixed(2);
  updateContinentLabels();
  updatePresetHelp();
}

function updateContinentLabels(){
  $('fragmentationValue').textContent=Number($('fragmentation').value).toFixed(2);
  $('separationValue').textContent=Number($('separation').value).toFixed(2);
}

function updateWorldProfileControls(){
  const preset=$('worldPreset').value;
  const custom=preset==='custom';
  const legacy=preset==='legacy';
  $('parameterCsvLabel').hidden=!custom;
  $('profileHelp').textContent=legacy
    ? 'Legacy rc.1 模型继续按创建时旧规则读取；Tester 不会把它伪装成 v1.1 Profile。'
    : custom
      ? 'Custom 使用下方完整 25 参数快照；这份快照会进入 .geoworld 并锁定。'
      : '当前使用 '+$('worldPreset').selectedOptions[0].textContent+' 参数组；Preview、Minecraft 与验收读取同一锁定 Profile。';
}

['width','height','cellSize'].forEach(id=>$(id).addEventListener('input',updateRangePreview));
['ocean','fragmentation','separation'].forEach(id=>$(id).addEventListener('input',()=>{
  updateContinentLabels();
  syncContinentPreset();
}));
$('continentPreset').addEventListener('change',()=>applyContinentPreset($('continentPreset').value));
$('worldPreset').addEventListener('change',()=>{
  if($('worldPreset').value!=='custom')$('parameterCsv').value='';
  updateWorldProfileControls();
});
$('modelName').addEventListener('input',()=>{modelNameTouched=true});
$('exportName').addEventListener('input',()=>{exportNameTouched=true});
$('randomSeed').addEventListener('click',()=>{
  const values=new Uint32Array(1);
  crypto.getRandomValues(values);
  $('seed').value=String((values[0]&0x7fffffff)||1);
});

$('generate').onclick=async()=>{
  const previous=active;
  setBusy(true);
  clearExportResult();
  $('progressBar').style.width='0%';
  $('progressText').textContent='开始生成…';
  $('modelMessage').textContent='';
  if(previous){
    setMapNotice(`正在生成新世界；当前地图暂时仍是 Seed ${previous.seed} 的上一次成功结果。`,'warn');
  }else{
    setMapNotice('正在生成世界…','warn');
  }
  poll=setInterval(refreshStatus,250);
  try{
    const next=await api('/api/generate',{method:'POST',body:JSON.stringify({
      seed:Number($('seed').value),
      width:Number($('width').value),
      height:Number($('height').value),
      cellSize:Number($('cellSize').value),
      plateCount:Number($('plates').value),
      oceanFraction:Number($('ocean').value),
      continentFragmentation:Number($('fragmentation').value),
      continentSeparation:Number($('separation').value),
      worldPreset:$('worldPreset').value,
      parameters:$('worldPreset').value==='custom' ? $('parameterCsv').value.trim() : ''
    })});
    active=next;
    acceptanceData=null;
    previewFeatures=null;
    resetMineralPreview();
    resetStructurePreview();
    clearAcceptanceView();
    resetMapView();
    $('progressBar').style.width='100%';
    $('progressText').textContent=`生成完成 · ${(active.durationMs/1000).toFixed(2)}s`;
    $('topStatus').textContent='已生成';
    if(!modelNameTouched||!$('modelName').value.trim()){
      $('modelName').value='seed-'+active.seed;
      modelNameTouched=false;
    }
    if(!exportNameTouched||!$('exportName').value.trim()){
      $('exportName').value='GeoWorld '+active.seed+' QA';
      exportNameTouched=false;
    }
    clearSample();
    renderStats();
    await loadMap();
    await refreshFeatures();
    renderHomeCurrentWorld();
    await refreshModels();
  }catch(e){
    $('progressText').innerHTML='<span class="error">'+escapeHtml(e.message)+'</span>';
    $('topStatus').textContent=e.network?'服务断开':'生成失败';
    if(previous){
      setMapNotice(`生成失败；当前地图仍是 Seed ${previous.seed} 的上一次成功结果。`,'error');
    }else{
      setMapNotice('生成失败，没有可显示的新地图。','error');
    }
  }finally{
    clearInterval(poll);
    poll=null;
    setBusy(false);
  }
};

function selectLayer(next,{load=true}={}){
  layer=next;
  document.querySelectorAll('.layer').forEach(b=>b.classList.toggle('active',b.dataset.layer===next));
  updateLayerHelp();
  if(load&&active)void loadMap();
}

document.querySelectorAll('.layer').forEach(btn=>btn.onclick=()=>{
  if(btn.disabled)return;
  selectLayer(btn.dataset.layer);
});

function mapRenderSize(){
  const wrap=$('mapWrap').getBoundingClientRect();
  const aspect=Math.max(.2,active.geometry.SpanXBlocks/active.geometry.SpanZBlocks);
  const dpr=Math.min(2,window.devicePixelRatio||1);
  let width=Math.round(Math.max(1200,wrap.width*dpr*1.7));
  width=Math.min(2400,width);
  let height=Math.round(width/aspect);
  if(height>1600){height=1600;width=Math.min(2400,Math.round(height*aspect));}
  if(height<600){height=600;width=Math.min(2400,Math.round(height*aspect));}
  return {width,height};
}

async function loadMap(){
  if(!active)return;
  const token=++mapLoadToken;
  const targetSeed=active.seed;
  const oldSeed=renderedSeed;
  setMapNotice(`正在渲染 ${layerName(layer)}…`,'warn');
  let r;
  try{
    const size=mapRenderSize();
    r=await fetch(`/api/layer?name=${encodeURIComponent(layer)}&width=${size.width}&height=${size.height}&t=${Date.now()}`);
  }catch(err){
    updateBackend(false);
    setMapNotice(oldSeed===null?'地图加载失败：后端已断开。':`地图加载失败；当前仍显示 Seed ${oldSeed} 的旧地图。`,'error');
    return;
  }
  updateBackend(true);
  if(!r.ok){
    const data=await r.json().catch(()=>({}));
    setMapNotice('地图加载失败：'+(data.error||('HTTP '+r.status)),'error');
    return;
  }
  const blob=await r.blob();
  if(token!==mapLoadToken)return;
  const nextURL=URL.createObjectURL(blob);
  if(mapObjectURL)URL.revokeObjectURL(mapObjectURL);
  mapObjectURL=nextURL;
  $('map').src=nextURL;
  $('mapWrap').classList.remove('empty');
  renderedSeed=targetSeed;
  setMapNotice('');
}

function layerName(v){
  return ({
    final:'地形',
    biome:'群系',
    landform:'地貌',
    water:'水体',
    surface:'地表',
    minerals:'矿产分区',
    tectonic:'地质构造'
  })[v]||v;
}

function updateLayerHelp(){
  const help={
    final:'地形：按最终 Runtime 高度、水体和海岸渲染整个 World Model。',
    biome:'群系：连续显示 GeoWorld 最终 Minecraft 群系家族，不是抽样点。',
    landform:'地貌：显示平原、丘陵、山麓、山地、高原、洪泛平原、峡谷等连续分区。',
    water:'水体：显示海洋、湖泊与河流的最终 Runtime 判定；河流/湖泊几何还会作为独立要素覆盖在地图上。',
    surface:'地表：显示森林土壤、草地、沙漠、湿地、裸岩、雪地等最终地表方案。',
    minerals:'矿产分区：按地形和生态条件显示各区域的主要矿产倾向；这是地质分区，不伪装成单个矿脉坐标。',
    tectonic:'地质构造：显示 GeoWorld 自身的造山带、裂谷、岛弧、海沟、洋中脊等构造场。'
  };
  $('layerHelp').textContent=help[layer]||'';
}

$('mapZoomIn').onclick=()=>zoomMapCentered(1.25);
$('mapZoomOut').onclick=()=>zoomMapCentered(0.8);
$('mapZoomReset').onclick=resetMapView;
$('mobileZoomIn').onclick=()=>zoomMapCentered(1.3);
$('mobileZoomOut').onclick=()=>zoomMapCentered(0.77);

function worldPointFromClient(clientX,clientY){
  if(!active)return null;
  const rect=$('map').getBoundingClientRect();
  if(rect.width<=0||rect.height<=0)return null;
  const fx=(clientX-rect.left)/rect.width;
  const fz=(clientY-rect.top)/rect.height;
  if(fx<0||fx>1||fz<0||fz>1)return null;
  const g=active.geometry;
  return {x:g.OriginX+fx*g.SpanXBlocks,z:g.OriginZ+fz*g.SpanZBlocks};
}

function centerMapOnWorld(x,z,targetZoom=Math.max(4,mapZoom)){
  if(!active)return;
  const g=active.geometry;
  const fx=(x-g.OriginX)/g.SpanXBlocks;
  const fz=(z-g.OriginZ)/g.SpanZBlocks;
  if(fx<0||fx>1||fz<0||fz>1)return;
  resetMapView();
  const rect=$('map').getBoundingClientRect();
  const pointX=rect.left+fx*rect.width;
  const pointY=rect.top+fz*rect.height;
  zoomMapAt(pointX,pointY,targetZoom);
  const zoomed=$('map').getBoundingClientRect();
  const wrap=$('mapWrap').getBoundingClientRect();
  const nowX=zoomed.left+fx*zoomed.width;
  const nowY=zoomed.top+fz*zoomed.height;
  mapPanX+=wrap.left+wrap.width/2-nowX;
  mapPanY+=wrap.top+wrap.height/2-nowY;
  applyMapTransform();
}

$('jumpToCoord').onclick=async()=>{
  if(!active)return;
  const x=Number($('jumpX').value);
  const z=Number($('jumpZ').value);
  if(!Number.isFinite(x)||!Number.isFinite(z))return;
  centerMapOnWorld(x,z);
  await focusWorldPoint(x,z);
};
['jumpX','jumpZ'].forEach(id=>$(id).addEventListener('keydown',e=>{
  if(e.key==='Enter')$('jumpToCoord').click();
}));

$('mapWrap').addEventListener('wheel',e=>{
  if(!active||$('mapWrap').classList.contains('empty'))return;
  e.preventDefault();
  const delta=Math.max(-140,Math.min(140,e.deltaY));
  const factor=Math.exp(-delta*0.0017);
  zoomMapAt(e.clientX,e.clientY,mapZoom*factor);
},{passive:false});

$('mapWrap').addEventListener('dblclick',e=>{
  if(!active||$('mapWrap').classList.contains('empty'))return;
  e.preventDefault();
  zoomMapAt(e.clientX,e.clientY,mapZoom*1.7);
});

function pointerDistance(){
  const pts=[...touchPoints.values()];
  if(pts.length<2)return 0;
  return Math.hypot(pts[0].x-pts[1].x,pts[0].y-pts[1].y);
}
function pointerMidpoint(){
  const pts=[...touchPoints.values()];
  if(pts.length<2)return null;
  return {x:(pts[0].x+pts[1].x)/2,y:(pts[0].y+pts[1].y)/2};
}

$('mapWrap').addEventListener('pointerdown',e=>{
  if(!active||$('mapWrap').classList.contains('empty'))return;
  if(e.pointerType==='mouse'&&e.button!==0)return;
  if(e.target.closest?.('.mobile-map-actions'))return;
  touchPoints.set(e.pointerId,{x:e.clientX,y:e.clientY});
  try{$('mapWrap').setPointerCapture(e.pointerId)}catch(_){}
  if(touchPoints.size>=2){
    pinchStartDistance=pointerDistance();
    pinchStartZoom=mapZoom;
    mapDragging=false;
    $('mapWrap').classList.add('dragging');
    e.preventDefault();
    return;
  }
  if(mapZoom>1.001){
    mapDragging=true;
    mapDragMoved=false;
    mapDragStartX=e.clientX;
    mapDragStartY=e.clientY;
    mapPanStartX=mapPanX;
    mapPanStartY=mapPanY;
    $('mapWrap').classList.add('dragging');
    e.preventDefault();
  }
});

$('mapWrap').addEventListener('pointermove',e=>{
  if(touchPoints.has(e.pointerId))touchPoints.set(e.pointerId,{x:e.clientX,y:e.clientY});
  if(touchPoints.size>=2&&pinchStartDistance>0){
    const distance=pointerDistance();
    const mid=pointerMidpoint();
    if(mid&&distance>0){
      zoomMapAt(mid.x,mid.y,pinchStartZoom*(distance/pinchStartDistance));
      mapDragMoved=true;
    }
    e.preventDefault();
    return;
  }
  if(!mapDragging)return;
  const dx=e.clientX-mapDragStartX;
  const dy=e.clientY-mapDragStartY;
  if(Math.hypot(dx,dy)>3)mapDragMoved=true;
  mapPanX=mapPanStartX+dx;
  mapPanY=mapPanStartY+dy;
  applyMapTransform();
  e.preventDefault();
});

function finishMapPointer(e){
  touchPoints.delete(e.pointerId);
  try{$('mapWrap').releasePointerCapture(e.pointerId)}catch(_){}
  if(touchPoints.size>=2){
    pinchStartDistance=pointerDistance();
    pinchStartZoom=mapZoom;
    return;
  }
  if(touchPoints.size===1&&mapZoom>1.001){
    const p=[...touchPoints.values()][0];
    mapDragging=true;
    mapDragStartX=p.x;
    mapDragStartY=p.y;
    mapPanStartX=mapPanX;
    mapPanStartY=mapPanY;
    return;
  }
  mapDragging=false;
  pinchStartDistance=0;
  $('mapWrap').classList.remove('dragging');
  if(mapDragMoved){
    suppressMapClick=true;
    setTimeout(()=>{suppressMapClick=false},0);
  }
  updateActionState();
}
$('mapWrap').addEventListener('pointerup',finishMapPointer);
$('mapWrap').addEventListener('pointercancel',finishMapPointer);

$('mapWrap').addEventListener('click',async e=>{
  if(e.target.closest?.('.mobile-map-actions'))return;
  if(suppressMapClick){
    suppressMapClick=false;
    return;
  }
  const p=worldPointFromClient(e.clientX,e.clientY);
  if(!p)return;
  await focusWorldPoint(p.x,p.z);
});
$('save').onclick=async()=>{
  $('modelMessage').textContent='';
  try{
    const r=await api('/api/save',{method:'POST',body:JSON.stringify({name:$('modelName').value})});
    $('modelMessage').innerHTML='<span class="ok">已保存：'+escapeHtml(r.name)+'.geoworld</span>';
    await refreshModels(r.name);
  }catch(e){
    $('modelMessage').innerHTML='<span class="error">'+escapeHtml(e.message)+'</span>';
  }
};

$('load').onclick=async()=>{
  const name=$('models').value;
  if(!name)return;
  setBusy(true);
  clearExportResult();
  $('modelMessage').textContent='';
  try{
    active=await api('/api/load',{method:'POST',body:JSON.stringify({name})});
    acceptanceData=null;
    previewFeatures=null;
    resetMineralPreview();
    resetStructurePreview();
    clearAcceptanceView();
    resetMapView();
    $('seed').value=active.seed;
    $('width').value=active.geometry.Width;
    $('height').value=active.geometry.Height;
    const cell=active.geometry.Width>1?active.geometry.SpanXBlocks/(active.geometry.Width-1):0;
    if(cell===64||cell===128)$('cellSize').value=String(cell);
    $('plates').value=active.plateCount;
    $('ocean').value=active.oceanFraction;
    $('fragmentation').value=active.continentFragmentation||0.35;
    $('separation').value=active.continentSeparation||0.65;
    $('worldPreset').value=active.worldPreset||'standard';
    if(!$('worldPreset').querySelector(`option[value="${$('worldPreset').value}"]`)){
      $('worldPreset').value='custom';
    }
    $('parameterCsv').value=active.worldPreset==='legacy' ? '' : (active.parameterCsv||'');
    updateWorldProfileControls();
    updateRangePreview();
    updateContinentLabels();
    syncContinentPreset();
    if(!active.intermediateDiagnostics&&(layer==='raw'||layer==='coast')){
      selectLayer('final',{load:false});
    }
    $('topStatus').textContent='已重开';
    $('progressText').textContent=`模型重开完成 · ${active.durationMs}ms`;
    $('modelName').value=name;
    modelNameTouched=true;
    if(!exportNameTouched||!$('exportName').value.trim()){
      $('exportName').value=name+' QA';
      exportNameTouched=false;
    }
    $('modelMessage').innerHTML='<span class="warn">当前是已保存模型的只读重开结果；如需再次保存，请重新生成世界。</span>';
    clearSample();
    renderStats();
    await loadMap();
    await refreshFeatures();
    renderHomeCurrentWorld();
  }catch(e){
    $('modelMessage').innerHTML='<span class="error">'+escapeHtml(e.message)+'</span>';
  }finally{
    setBusy(false);
  }
};

$('exportMinecraft').onclick=async()=>{
  if(!active)return;
  setBusy(true);
  lastExport=null;
  $('openExport').hidden=true;
  $('exportMessage').innerHTML='<span class="warn">正在编译 Java writer 并导出存档…</span>';
  $('topStatus').textContent='导出中';
  try{
    const radius=Math.max(0,Math.min(31,Number($('exportRadius').value)||0));
    $('exportRadius').value=String(radius);
    const result=await api('/api/export-minecraft',{
      method:'POST',
      body:JSON.stringify({
        name:$('exportName').value,
        directory:$('exportDir').value,
        radius,
        qaMode:$('exportQA').checked,
        overwrite:$('exportOverwrite').checked
      })
    });
    lastExport=result;
    $('openExport').hidden=false;
    $('topStatus').textContent='已导出';
    $('exportMessage').innerHTML=
      '<span class="ok">导出完成</span><br>'+
      'Minecraft '+escapeHtml(result.minecraftVersion)+
      ' · '+Number(result.chunks).toLocaleString()+' Chunk'+
      ' · '+Number(result.regions).toLocaleString()+' Region<br>'+
      '出生点 '+Number(result.spawnX)+', '+Number(result.spawnY)+', '+Number(result.spawnZ)+
      (result.qaMode?' · QA 创造/作弊':' · 生存模式')+
      '<br><span class="muted">'+escapeHtml(result.output)+'</span>';
    renderExportOverlay();
  }catch(e){
    $('topStatus').textContent='导出失败';
    $('exportMessage').innerHTML='<span class="error">'+escapeHtml(e.message)+'</span>';
  }finally{
    setBusy(false);
  }
};

$('openExport').onclick=async()=>{
  if(!lastExport)return;
  try{
    await api('/api/open-folder',{
      method:'POST',
      body:JSON.stringify({path:lastExport.output})
    });
  }catch(e){
    $('exportMessage').innerHTML+='<br><span class="error">'+escapeHtml(e.message)+'</span>';
  }
};

async function refreshModels(selectName=''){
  const previous=selectName||$('models').value;
  try{
    const list=await api('/api/models');
    $('models').innerHTML=list.length
      ?list.map(m=>`<option value="${escapeAttr(m.name)}">${escapeHtml(m.name)} · Seed ${m.seed} · ${formatBytes(m.fileBytes)}</option>`).join('')
      :'<option value="">暂无</option>';
    if(previous&&list.some(m=>m.name===previous))$('models').value=previous;
  }catch(e){
    $('models').innerHTML='<option value="">读取失败</option>';
    if(e.network)$('modelMessage').innerHTML='<span class="error">'+escapeHtml(e.message)+'</span>';
  }
  updateActionState();
}


async function focusWorldPoint(x,z){
  if(!active)return;
  lastFocusPoint={x,z};
  renderCursorOverlay();
  try{
    renderSample(await api('/api/sample?x='+encodeURIComponent(x)+'&z='+encodeURIComponent(z)));
  }catch(err){
    $('sampleEmpty').hidden=true;
    $('sample').hidden=false;
    $('sample').innerHTML='<div class="error-card">'+escapeHtml(err.message)+'</div>';
  }
}

function renderEmptyStats(){
  $('stats').outerHTML='<div id="stats" class="kv"><span>Seed</span><b>—</b></div>';
}

function renderStats(){
  if(!active)return;
  const g=active.geometry;
  const cell=g.Width>1?g.SpanXBlocks/(g.Width-1):0;
  $('stats').outerHTML=`<div id="stats">
    ${kv('来源',active.source==='generated'?'刚生成':'模型重开')}
    ${kv('Seed',active.seed)}
    ${kv('网格',g.Width+' × '+g.Height)}
    ${kv('采样间距',Math.round(cell)+' 方块 / 点')}
    ${kv('世界范围',Math.round(g.SpanXBlocks)+' × '+Math.round(g.SpanZBlocks)+' 方块')}
    ${kv('板块数',active.plateCount)}
    ${kv('洋壳比例',Number(active.oceanFraction).toFixed(2))}
    ${kv('大陆破碎度',Number(active.continentFragmentation).toFixed(2))}
    ${kv('大陆分离度',Number(active.continentSeparation).toFixed(2))}
    ${kv('世界预设',active.worldPreset||'legacy')}
    ${kv('Profile',active.worldProfileVersion||'—')}
    ${kv('生成规则',active.generationRulesVersion||'—')}
    ${kv('World Model',active.worldModelVersion||'—')}
    ${kv('参数快照',active.parameterSnapshotSha256 ? active.parameterSnapshotSha256.slice(0,16)+'…' : 'legacy')}
    ${kv('主要陆块',active.majorLandComponents)}
    ${kv('全部陆块',active.landComponents)}
    ${kv('最大陆块占比',(Number(active.largestLandShare)*100).toFixed(1)+'%')}
    ${kv('河段',active.riverSegments)}
    ${kv('湖泊节点',active.lakeNodes)}
    ${diagnosticKV('最终海岸冲突',active.finalCoastMismatch)}
    ${diagnosticKV('最终河床逆坡',active.finalRiverUpslope)}
    ${diagnosticKV('最终湖床违规',active.finalLakeBedViolations)}
    ${diagnosticKV('Runtime 节点回退',active.runtimeFallback)}
    ${diagnosticKV('Runtime 节点冲突',active.runtimeMismatch)}
    ${active.intermediateDiagnostics?`
      ${kv('海岸校正节点',active.coastAdjustedNodes)}
      ${kv('Raw 海陆冲突',active.surfaceMismatchBefore)}
      ${kv('约束后海陆冲突',active.surfaceMismatchAfter)}
      ${kv('应用后海陆冲突',active.appliedCoastSurfaceMismatch)}
      ${kv('Raw 河流逆坡',active.rawUpslopeRiverEdges)}
      ${kv('目标河流逆坡',active.targetUpslopeRiverEdges)}
      ${kv('海岸后湖床待挖',active.lakeBedViolations)}
    `:''}
  </div>`;
}

function clearSample(){
  lastFocusPoint=null;
  $('cursor').style.display='none';
  $('sampleEmpty').hidden=false;
  $('sample').hidden=true;
  $('sample').innerHTML='';
}

function renderSample(s){
  $('sampleEmpty').hidden=true;
  $('sample').hidden=false;
  const water=s.waterSurfaceDefined
    ?`${s.waterKind} · 水面 ${f(s.waterSurfaceElevation)} · 深度 ${f(s.waterDepth)}`
    :s.waterKind;
  $('sample').innerHTML=`
    <div class="sample-section"><h3>最终地面 / 水体</h3>
      ${kv('坐标',Math.round(s.x)+', '+Math.round(s.z))}
      ${kv('地面高度',f(s.groundHeight))}
      ${kv('水体',water)}
      ${kv('海洋判定权重',f(s.oceanEligibility))}
      ${kv('河流权重',f(s.riverWeight))}
    </div>
    <div class="sample-section"><h3>生态 / Minecraft</h3>
      ${kv('生态区',s.ecoRegion)}
      ${kv('生物群系',s.biomeFamily)}
      ${kv('地形类型',s.terrainVariant)}
      ${kv('地表方案',s.surfaceProfile)}
      ${kv('植被方案',s.vegetationProfile)}
      ${kv('配置置信度',f(s.profileConfidence))}
    </div>
    <div class="sample-section"><h3>一致性检查</h3>
      ${kv('来源节点',s.sourceNode)}
      ${kv('来源是否陆地',s.sourceLand?'是':'否')}
      ${kv('语义回退',s.semanticFallback?'<span class="error">是</span>':'<span class="ok">否</span>')}
      ${kv('语义冲突',s.semanticMismatch?'<span class="error">是</span>':'<span class="ok">否</span>')}
    </div>`;
}

function kv(k,v){return `<div class="kv"><span>${k}</span><b>${v}</b></div>`}
function diagnosticKV(k,v){
  const n=Number(v)||0;
  return kv(k,n===0?'<span class="ok">0</span>':'<span class="error">'+n+'</span>');
}
function f(v){return Number(v).toFixed(6)}
function formatBytes(n){return n<1048576?(n/1024).toFixed(1)+' KB':(n/1048576).toFixed(2)+' MB'}
function escapeHtml(s){return String(s).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]))}
function escapeAttr(s){return escapeHtml(s)}


function navigatePage(page){
  currentPage=page;
  document.querySelectorAll('.page').forEach(el=>el.classList.toggle('active',el.id==='page-'+page));
  document.querySelectorAll('.nav-item').forEach(el=>el.classList.toggle('active',el.dataset.page===page));
  if(page==='world')requestAnimationFrame(()=>{renderExportOverlay();renderCursorOverlay();positionFeatureOverlay()});
  if(page==='acceptance'&&active&&!acceptanceData)void generateAcceptanceTable();
  if(page==='home')renderHomeCurrentWorld();
  history.replaceState(null,'','#'+page);
}

function renderHomeCurrentWorld(){
  const el=$('homeCurrentWorld');
  if(!active){
    el.className='empty-card';
    el.textContent='尚未生成或打开世界模型。';
    return;
  }
  el.className='panel-body';
  el.innerHTML=`
    <div class="about-kv">
      <span>Seed</span><b>${active.seed}</b>
      <span>预设</span><b>${escapeHtml(active.worldPreset||'standard')}</b>
      <span>Generation Rules</span><b>${escapeHtml(active.generationRulesVersion||'—')}</b>
      <span>河流 / 湖泊</span><b>${active.riverCount||0} / ${active.lakeCount||0}</b>
      <span>来源</span><b>${active.source==='generated'?'当前生成':'已保存模型'}</b>
    </div>`;
}

function clearAcceptanceView(){
  acceptanceData=null;
  $('acceptanceTableBody').innerHTML='<tr><td colspan="7" class="empty-cell">暂无验收表</td></tr>';
  $('acceptanceBigFilter').innerHTML='<option value="">全部大类</option>';
  $('acceptanceMessage').textContent=active?'点击“生成验收表”生成当前模型的验收清单。':'请先生成或打开一个 World Model。';
  renderAcceptanceSummary();
}

function acceptanceType(){
  return document.querySelector('input[name="acceptanceType"]:checked')?.value||'spot';
}

async function generateAcceptanceTable(){
  if(!active)return;
  const type=acceptanceType();
  $('acceptanceMessage').textContent='正在从当前 World Model 生成验收表…';
  $('acceptanceGenerate').disabled=true;
  try{
    acceptanceData=await api('/api/acceptance?type='+encodeURIComponent(type)+'&format=json');
    populateAcceptanceFilter();
    renderAcceptanceTable();
    $('acceptanceMessage').textContent=type==='full'
      ?'完整验收表来自 Acceptance Registry。'
      :'抽检表严格从完整验收表中按每个大类抽取 2 条。';
  }catch(e){
    acceptanceData=null;
    $('acceptanceMessage').innerHTML='<span class="error">'+escapeHtml(e.message)+'</span>';
    $('acceptanceTableBody').innerHTML='<tr><td colspan="7" class="empty-cell">生成失败</td></tr>';
  }finally{
    updateActionState();
  }
}

function normalizeAcceptanceRows(){
  if(!acceptanceData)return [];
  const rows=[];
  if(acceptanceData.format==='geoworld-spot-acceptance-table'){
    for(const big of acceptanceData.bigCategories||[]){
      for(const selected of big.items||[]){
        rows.push({bigId:big.id,bigName:big.name,smallId:selected.smallCategoryId,smallName:selected.smallCategoryName,item:selected.item});
      }
    }
    return rows;
  }
  for(const big of acceptanceData.bigCategories||[]){
    for(const small of big.smallCategories||[]){
      for(const item of small.items||[])rows.push({bigId:big.id,bigName:big.name,smallId:small.id,smallName:small.name,item});
    }
  }
  return rows;
}

function populateAcceptanceFilter(){
  const select=$('acceptanceBigFilter');
  const previous=select.value;
  select.innerHTML='<option value="">全部大类</option>';
  for(const big of acceptanceData?.bigCategories||[]){
    const option=document.createElement('option');
    option.value=big.id;
    option.textContent=big.name;
    select.appendChild(option);
  }
  if([...select.options].some(o=>o.value===previous))select.value=previous;
}

function acceptanceTP(item){
  if(!item.points||item.points.length===0)return '无 TP；按验收方式执行';
  return item.points.map(p=>p.tp).join(' · ');
}

function acceptanceStatusClass(status){
  if(status==='PASS')return 'pass';
  if(status==='FAIL')return 'fail';
  return 'manual';
}

function renderAcceptanceTable(){
  if(!acceptanceData){clearAcceptanceView();return}
  const filter=$('acceptanceBigFilter').value;
  const rows=normalizeAcceptanceRows().filter(row=>!filter||row.bigId===filter);
  $('acceptanceTableBody').innerHTML=rows.length?rows.map(row=>`
    <tr>
      <td>${escapeHtml(row.bigName)}</td>
      <td>${escapeHtml(row.smallName)}</td>
      <td><b>${escapeHtml(row.item.id)}</b></td>
      <td>${escapeHtml(row.item.name)}</td>
      <td>${escapeHtml('Seed '+acceptanceData.seed+'；'+acceptanceTP(row.item))}</td>
      <td>${escapeHtml(row.item.expected)}</td>
      <td><span class="status-badge ${acceptanceStatusClass(row.item.status)}">${escapeHtml(row.item.status)}</span></td>
    </tr>`).join(''):'<tr><td colspan="7" class="empty-cell">当前筛选下没有条目</td></tr>';
  renderAcceptanceSummary(rows);
}

function renderAcceptanceSummary(rows=null){
  const cards=$('acceptanceSummary').querySelectorAll('.summary-card b');
  if(!active){
    cards[0].textContent='未加载';cards[1].textContent=acceptanceType()==='full'?'完整表':'抽检表';cards[2].textContent='—';cards[3].textContent='—';
    return;
  }
  const allRows=acceptanceData?normalizeAcceptanceRows():[];
  cards[0].textContent='Seed '+active.seed;
  cards[1].textContent=acceptanceType()==='full'?'完整验收表':'抽检表';
  cards[2].textContent=acceptanceData?(acceptanceData.bigCategories||[]).length:'—';
  cards[3].textContent=acceptanceData?(rows||allRows).length:'—';
}

function downloadAcceptance(format){
  if(!active)return;
  const type=acceptanceType();
  const href='/api/acceptance?type='+encodeURIComponent(type)+'&format='+encodeURIComponent(format)+'&download=1';
  const a=document.createElement('a');
  a.href=href;
  a.download='';
  document.body.appendChild(a);
  a.click();
  a.remove();
}

function resetMineralPreview(){
  mineralBodies=null;
  mineralLoadToken++;
  mineralBridgeUnavailable=false;
  if(mineralRefreshTimer){
    clearTimeout(mineralRefreshTimer);
    mineralRefreshTimer=null;
  }
  const detail=$('mineralDetail');
  if(detail){detail.hidden=true;detail.textContent='';}
  const status=$('mineralStatus');
  if(status)status.textContent='放大到 5× 后加载主要矿体；10× 后切换详细矿体。小型散布矿体数量很大时会按当前视野上限显示。';
}

function mineralViewportBounds(){
  if(!active||$('mapWrap').classList.contains('empty'))return null;
  const img=$('map').getBoundingClientRect();
  const wrap=$('mapWrap').getBoundingClientRect();
  const left=Math.max(img.left,wrap.left);
  const right=Math.min(img.right,wrap.right);
  const top=Math.max(img.top,wrap.top);
  const bottom=Math.min(img.bottom,wrap.bottom);
  if(right<=left||bottom<=top||img.width<=0||img.height<=0)return null;
  const fx0=Math.max(0,Math.min(1,(left-img.left)/img.width));
  const fx1=Math.max(0,Math.min(1,(right-img.left)/img.width));
  const fz0=Math.max(0,Math.min(1,(top-img.top)/img.height));
  const fz1=Math.max(0,Math.min(1,(bottom-img.top)/img.height));
  const g=active.geometry;
  const pad=Math.max(96,Math.min(384,Math.max(
    (fx1-fx0)*g.SpanXBlocks,
    (fz1-fz0)*g.SpanZBlocks
  )*.04));
  return {
    minX:Math.floor(g.OriginX+fx0*g.SpanXBlocks-pad),
    maxX:Math.ceil(g.OriginX+fx1*g.SpanXBlocks+pad),
    minZ:Math.floor(g.OriginZ+fz0*g.SpanZBlocks-pad),
    maxZ:Math.ceil(g.OriginZ+fz1*g.SpanZBlocks+pad)
  };
}

function scheduleMineralRefresh(delay=260){
  if(mineralRefreshTimer)clearTimeout(mineralRefreshTimer);
  mineralRefreshTimer=setTimeout(()=>{
    mineralRefreshTimer=null;
    void refreshMineralBodies();
  },delay);
}

async function refreshMineralBodies(){
  const enabled=enabledFeatureKinds().has('minerals');
  if(!active||!enabled||mineralBridgeUnavailable){
    mineralBodies=null;
    renderFeatureOverlay();
    return;
  }
  if(mapZoom<5){
    mineralBodies=null;
    $('mineralStatus').textContent='继续放大到 5× 后显示真实规划矿体。';
    renderFeatureOverlay();
    return;
  }
  if(mapDragging||touchPoints.size>0){
    scheduleMineralRefresh(220);
    return;
  }
  const bounds=mineralViewportBounds();
  if(!bounds)return;
  const mode=mapZoom>=10?'all':'major';
  const limit=window.innerWidth<=900?900:1600;
  const token=++mineralLoadToken;
  $('mineralStatus').textContent=mode==='all'
    ? '正在加载当前视野的详细矿体…'
    : '正在加载当前视野的主要矿体…';
  try{
    const q=new URLSearchParams({
      minX:String(bounds.minX),maxX:String(bounds.maxX),
      minZ:String(bounds.minZ),maxZ:String(bounds.maxZ),
      mode,limit:String(limit)
    });
    const data=await api('/api/minerals?'+q.toString());
    if(token!==mineralLoadToken)return;
    mineralBodies=data;
    const suffix=data.truncated?'（数量较多，已从当前视野均匀抽样显示 '+Number(data.returned||0).toLocaleString()+' 个）':'';
    $('mineralStatus').textContent=
      (mode==='all'?'详细矿体 ':'主要矿体 ')+
      Number(data.total||0).toLocaleString()+' 个'+suffix;
    renderFeatureOverlay();
  }catch(e){
    if(token!==mineralLoadToken)return;
    mineralBodies=null;
    if(e.status===503){
      mineralBridgeUnavailable=true;
      $('mineralStatus').textContent='真实矿脉需要从可执行 GeoWorld JAR 启动 Web 工具。';
    }else if(e.status===413){
      $('mineralStatus').textContent='当前视野仍太大，请继续放大后查看真实矿脉。';
    }else{
      $('mineralStatus').textContent='矿脉加载失败：'+e.message;
    }
    renderFeatureOverlay();
  }
}

function resetStructurePreview(){
  structureCandidates=null;
  structureLoadToken++;
  if(structureRefreshTimer){
    clearTimeout(structureRefreshTimer);
    structureRefreshTimer=null;
  }
  const detail=$('structureDetail');
  if(detail){detail.hidden=true;detail.textContent='';}
  const status=$('structureStatus');
  if(status)status.textContent='低倍缩放按空间区域合并结构，只指示大概分布；16× / 32× / 48× 仍保持高精度聚合；缩放越大，区域越小、结构类型越精确；64× 最大缩放展开当前视野全部候选。';
}

function scheduleStructureRefresh(delay=300){
  if(structureRefreshTimer)clearTimeout(structureRefreshTimer);
  structureRefreshTimer=setTimeout(()=>{
    structureRefreshTimer=null;
    void refreshStructureCandidates();
  },delay);
}

async function refreshStructureCandidates(){
  const enabled=enabledFeatureKinds().has('structures');
  if(!active||!enabled){
    structureCandidates=null;
    renderFeatureOverlay();
    return;
  }
  if(mapDragging||touchPoints.size>0){
    scheduleStructureRefresh(240);
    return;
  }
  const bounds=mineralViewportBounds();
  if(!bounds)return;
  const mode=mapZoom>=8?'all':'major';
  const limit=window.innerWidth<=1250?25000:50000;
  const token=++structureLoadToken;
  $('structureStatus').textContent=mode==='all'?'正在加载当前视野的详细结构候选…':'正在加载当前视野的主要结构分布…';
  try{
    const q=new URLSearchParams({
      minX:String(bounds.minX),maxX:String(bounds.maxX),
      minZ:String(bounds.minZ),maxZ:String(bounds.maxZ),
      mode,limit:String(limit)
    });
    const data=await api('/api/structures?'+q.toString());
    if(token!==structureLoadToken)return;
    structureCandidates=data;
    const structureSuffix=data.truncated
      ? '（共 '+Number(data.total||0).toLocaleString()+' 个，已按结构类别与视野位置均匀显示 '+Number(data.returned||0).toLocaleString()+' 个）'
      : Number(data.total||0).toLocaleString()+' 个';
    const visibleTypes=new Set((data.candidates||[]).map(candidate=>candidate.setId)).size;
    $('structureStatus').textContent=
      structureSuffix+' · '+visibleTypes+' 种结构 · '+
      (mapZoom>=FULL_DETAIL_ZOOM-.001
        ? '最大缩放：当前视野全部展开'
        : mapZoom<4
          ? '概略显示：按空间区域显示少量真实代表位置'
          : '聚合显示：继续放大可逐步提高位置精度')+
      (data.skippedStronghold?' · 强要塞不可用':' · 含强要塞');
    renderFeatureOverlay();
  }catch(e){
    if(token!==structureLoadToken)return;
    structureCandidates=null;
    $('structureStatus').textContent=e.status===413
      ? '当前视野仍太大，请继续放大后查看结构候选。'
      : '结构候选加载失败：'+e.message;
    renderFeatureOverlay();
  }
}

async function refreshFeatures(){
  if(!active){
    previewFeatures=null;
    renderFeatureOverlay();
    return;
  }
  try{
    previewFeatures=await api('/api/features');
    const c=previewFeatures.counts||{};
    $('featureNote').textContent=(previewFeatures.note||'')+
      ' 河流 '+Number(c.rivers||0).toLocaleString()+
      ' · 湖泊 '+Number(c.lakes||0).toLocaleString()+
      ' · 山峰 '+Number(c.peaks||0).toLocaleString();
    $('mapFeatureSummary').textContent=
      '河流 '+Number(c.rivers||0).toLocaleString()+
      ' · 湖泊 '+Number(c.lakes||0).toLocaleString()+
      ' · 山峰 '+Number(c.peaks||0).toLocaleString();
    renderFeatureOverlay();
  }catch(e){
    previewFeatures=null;
    $('featureNote').textContent='模型要素加载失败：'+e.message;
    $('mapFeatureSummary').textContent='模型要素加载失败';
    renderFeatureOverlay();
  }
}

function enabledFeatureKinds(){
  return new Set([...document.querySelectorAll('.feature-toggle:checked')].map(el=>el.dataset.feature));
}

function worldToOverlay(x,z){
  const g=active.geometry;
  return {
    x:(x-g.OriginX)/g.SpanXBlocks*1000,
    y:(z-g.OriginZ)/g.SpanZBlocks*1000
  };
}

function mineralName(kind){
  return ({
    COAL:'煤矿',IRON:'铁矿',COPPER:'铜矿',GOLD:'金矿',
    LAPIS:'青金石',REDSTONE:'红石',DIAMOND:'钻石',EMERALD:'绿宝石'
  })[kind]||kind;
}

function mineralSourceName(source){
  return ({
    VEIN:'常规矿体',GUARANTEE:'保底矿体',
    SCATTER:'散布矿体',OUTCROP:'地表露头'
  })[source]||source;
}

function showMineralDetail(body){
  const el=$('mineralDetail');
  el.hidden=false;
  el.textContent=
    mineralName(body.mineral)+' · '+mineralSourceName(body.source)+
    (body.large?' · 大型矿体':'')+
    (body.outcrop?' · 可形成露头':'')+
    ' · 中心 '+Math.round(body.x)+', Y '+Math.round(body.y)+', '+Math.round(body.z)+
    ' · 长约 '+body.length.toFixed(1)+' · 半径约 '+body.radius.toFixed(1);
}

function structureSetName(id){
  return ({
    'minecraft:abandoned_camp':'废弃营地',
    'minecraft:ancient_cities':'远古城市',
    'minecraft:buried_treasures':'埋藏宝藏',
    'minecraft:desert_pyramids':'沙漠神殿',
    'minecraft:igloos':'雪屋',
    'minecraft:jungle_temples':'丛林神庙',
    'minecraft:mineshafts':'废弃矿井',
    'minecraft:ocean_monuments':'海底神殿',
    'minecraft:ocean_ruins':'海底废墟',
    'minecraft:pillager_outposts':'掠夺者前哨站',
    'minecraft:ruined_portals':'废弃传送门',
    'minecraft:shipwrecks':'沉船',
    'minecraft:swamp_huts':'沼泽小屋',
    'minecraft:trail_ruins':'古迹废墟',
    'minecraft:trial_chambers':'试炼密室',
    'minecraft:villages':'村庄',
    'minecraft:woodland_mansions':'林地府邸',
    'minecraft:strongholds':'强要塞'
  })[id]||id;
}

function structureVariantName(id){
  const exact={
    'minecraft:ancient_city':'远古城市',
    'minecraft:bastion_remnant':'堡垒遗迹',
    'minecraft:buried_treasure':'埋藏宝藏',
    'minecraft:desert_pyramid':'沙漠神殿',
    'minecraft:end_city':'末地城',
    'minecraft:fortress':'下界要塞',
    'minecraft:igloo':'雪屋',
    'minecraft:jungle_pyramid':'丛林神庙',
    'minecraft:mansion':'林地府邸',
    'minecraft:mineshaft':'废弃矿井',
    'minecraft:mineshaft_mesa':'恶地废弃矿井',
    'minecraft:monument':'海底神殿',
    'minecraft:nether_fossil':'下界化石',
    'minecraft:ocean_ruin_cold':'冷水海底废墟',
    'minecraft:ocean_ruin_warm':'暖水海底废墟',
    'minecraft:pillager_outpost':'掠夺者前哨站',
    'minecraft:ruined_portal':'废弃传送门',
    'minecraft:ruined_portal_desert':'沙漠废弃传送门',
    'minecraft:ruined_portal_jungle':'丛林废弃传送门',
    'minecraft:ruined_portal_mountain':'山地废弃传送门',
    'minecraft:ruined_portal_nether':'下界废弃传送门',
    'minecraft:ruined_portal_ocean':'海洋废弃传送门',
    'minecraft:ruined_portal_swamp':'沼泽废弃传送门',
    'minecraft:shipwreck':'沉船',
    'minecraft:shipwreck_beached':'搁浅沉船',
    'minecraft:stronghold':'强要塞',
    'minecraft:swamp_hut':'沼泽小屋',
    'minecraft:trail_ruins':'古迹废墟',
    'minecraft:trial_chambers':'试炼密室',
    'minecraft:village_desert':'沙漠村庄',
    'minecraft:village_plains':'平原村庄',
    'minecraft:village_savanna':'热带草原村庄',
    'minecraft:village_snowy':'雪原村庄',
    'minecraft:village_taiga':'针叶林村庄'
  };
  if(exact[id])return exact[id];
  if(id.startsWith('minecraft:abandoned_camp_')){
    const biome=id.slice('minecraft:abandoned_camp_'.length);
    const biomeName=({
      bamboo_jungle:'竹林',birch_forest:'白桦林',cherry_grove:'樱花树林',
      dappled_forest:'斑驳森林',flower_forest:'繁花森林',forest:'森林',
      meadow:'草甸',old_growth_birch_forest:'原始白桦林',
      old_growth_pine_taiga:'原始松木针叶林',old_growth_spruce_taiga:'原始云杉针叶林',
      pale_garden:'苍白花园',savanna:'热带草原',snowy_taiga:'积雪针叶林',
      sparse_jungle:'稀疏丛林',swamp:'沼泽',taiga:'针叶林',
      windswept_forest:'风袭森林',wooded_badlands:'繁茂恶地'
    })[biome]||'未知群系';
    return biomeName+'废弃营地';
  }
  return '未知结构';
}

function structureMarkerKind(id){
  if(id==='minecraft:strongholds')return 'stronghold';
  if(['minecraft:villages','minecraft:abandoned_camp','minecraft:pillager_outposts','minecraft:woodland_mansions'].includes(id))return 'settlement';
  if(['minecraft:ocean_monuments','minecraft:ocean_ruins','minecraft:shipwrecks'].includes(id))return 'ocean';
  if(['minecraft:ancient_cities','minecraft:mineshafts','minecraft:trial_chambers'].includes(id))return 'underground';
  if(id==='minecraft:ruined_portals')return 'portal';
  if(id==='minecraft:buried_treasures')return 'treasure';
  return 'landmark';
}

function structureGlyphPath(id){
  return ({
    'geoworld:mixed_structures':'M-5-5 H-1 V-1 H-5 Z M1-5 H5 V-1 H1 Z M-5 1 H-1 V5 H-5 Z M1 1 H5 V5 H1 Z',
    'minecraft:villages':'M-6 1 L0-5 L6 1 M-4 0 V5 H4 V0 M-1 5 V1 H2 V5',
    'minecraft:abandoned_camp':'M-6 5 L0-5 L6 5 Z M0-5 V5',
    'minecraft:pillager_outposts':'M-5-4 H5 M-4-4 V5 H4 V-4 M-6 5 H6 M-2-1 H2',
    'minecraft:woodland_mansions':'M-6 5 V0 L0-5 L6 0 V5 M-4 5 V1 H4 V5 M-2 1 V5 M2 1 V5',
    'minecraft:ocean_monuments':'M-6 4 H6 M-5 4 V0 L0-4 L5 0 V4 M-3 1 H3 M0-4 V4',
    'minecraft:ocean_ruins':'M-6 5 H6 M-5 5 V-1 H-1 V5 M1 5 V-4 H5 V5',
    'minecraft:shipwrecks':'M-6 1 H6 L3 5 H-3 Z M0 1 V-5 M0-4 L4-1 H0',
    'minecraft:ancient_cities':'M-6 5 V1 L-3-2 L0 1 L3-4 L6 1 V5 M-4 5 V2 M0 5 V2 M4 5 V2',
    'minecraft:mineshafts':'M-6 5 V0 A6 6 0 0 1 6 0 V5 M-4 5 L-1 0 M4 5 L1 0 M-5 3 H5',
    'minecraft:trial_chambers':'M-5-5 H5 V5 H-5 Z M0-5 V5 M-5 0 H5 M-2-2 H2 V2 H-2 Z',
    'minecraft:ruined_portals':'M-5 5 V-5 H5 V5 H2 V-2 H-2 V5 Z',
    'minecraft:buried_treasures':'M-6-1 H6 V5 H-6 Z M-5-4 H5 L6-1 H-6 Z M-1 1 H1 V3 H-1 Z',
    'minecraft:desert_pyramids':'M-6 5 L0-5 L6 5 Z M-3 5 L0 0 L3 5 M-1 5 V2 H1 V5',
    'minecraft:igloos':'M-6 4 A6 6 0 0 1 6 4 H-6 M2 4 V1 H6 V4',
    'minecraft:jungle_temples':'M-6 5 H6 M-5 5 V2 H-3 V-1 H0 V-4 H3 V-1 H5 V5',
    'minecraft:swamp_huts':'M-6 0 L0-4 L6 0 M-5 0 H5 V4 H-5 Z M-4 4 V6 M4 4 V6',
    'minecraft:trail_ruins':'M-6 5 H6 M-5 5 V2 H-1 V-1 H3 V-4 H6 M-3 2 H1 M1-1 H5',
    'minecraft:strongholds':'M-6 5 V-2 H-3 V-5 H0 V-2 H3 V-5 H6 V5 M-6 1 H6 M-2 5 V1 H2 V5'
  })[id]||'M-5 5 H5 V-1 L0-5 L-5-1 Z';
}

function appendStructureGlyph(parent,id,scaleX,scaleY){
  const ns='http://www.w3.org/2000/svg';
  const glyph=document.createElementNS(ns,'path');
  glyph.setAttribute('d',structureGlyphPath(id));
  glyph.setAttribute('class','structure-glyph');
  glyph.setAttribute('transform',`scale(${scaleX} ${scaleY})`);
  parent.appendChild(glyph);
}

function openStructureInspector(){
  if(window.matchMedia('(max-width: 900px)').matches)setMobileSheet('features',true);
}

function showStructureDetail(candidate){
  const el=$('structureDetail');
  el.hidden=false;
  const meta=structureCandidates?.sets?.[candidate.setId]||{};
  const variants=(meta.structures||[]).map(structureVariantName).join(' / ');
  const stronghold=candidate.setId==='minecraft:strongholds';
  el.innerHTML=
    '<div class="structure-detail-title">'+escapeHtml(structureSetName(candidate.setId))+'</div>'+
    '<div class="structure-detail-grid">'+
      '<span>坐标</span><b>'+escapeHtml(candidate.x+', '+candidate.z)+'</b>'+
      '<span>区块坐标</span><b>'+escapeHtml(candidate.chunkX+', '+candidate.chunkZ)+'</b>'+
      (variants?'<span>具体结构</span><b>'+escapeHtml(variants)+'</b>':'')+
      '<span>位置性质</span><b>'+(stronghold?'精确规划位置':'原版结构放置候选位置')+'</b>'+
    '</div>'+
    '<div class="structure-detail-note">'+
      (stronghold
        ? '使用 Minecraft 26.3 原版同心环规则与 GeoWorld 群系偏移规则，并经过原生实现逐点一致性校验。'
        : '这里表示原版结构放置候选位置；最终是否实际生成，还需要通过群系和结构生成条件检查。')+
    '</div>';
  openStructureInspector();
}

function showStructureClusterDetail(cluster){
  const candidate=cluster.first;
  const el=$('structureDetail');
  el.hidden=false;
  const typeEntries=Object.entries(cluster.typeCounts||{})
    .sort((a,b)=>b[1]-a[1])
    .slice(0,5);
  const mixed=(cluster.typeCount||typeEntries.length)>1;
  el.innerHTML=
    '<div class="structure-detail-title">'+escapeHtml(mixed?'结构聚合':structureSetName(candidate.setId))+'</div>'+
    '<div class="structure-detail-grid">'+
      '<span>当前聚合</span><b>'+Number(cluster.count).toLocaleString()+' 个候选</b>'+
      '<span>代表位置</span><b>'+Math.round(cluster.worldX)+', '+Math.round(cluster.worldZ)+'</b>'+
      '<span>结构类型</span><b>'+escapeHtml(
        mixed
          ? typeEntries.map(([id,count])=>structureSetName(id)+' × '+count).join(' / ')+(cluster.typeCount>typeEntries.length?' / …':'')
          : structureSetName(candidate.setId)
      )+'</b>'+
      '<span>显示精度</span><b>'+(
        mapZoom<5?'概略区域':
        mapZoom<10?'区域分类':
        mapZoom<16?'细分区域':
        mapZoom<32?'高精度区域':
        mapZoom<48?'精细区域':'极精细区域'
      )+'</b>'+
    '</div>'+
    '<div class="structure-detail-note">图标落在该聚合区域内的真实候选位置，不使用几何平均产生虚假坐标；继续放大会缩小聚合范围，64× 时全部展开。</div>';
  openStructureInspector();
}

function overlayUnitScale(){
  const rect=$('map').getBoundingClientRect();
  return {
    x:rect.width>0?1000/rect.width:1,
    y:rect.height>0?1000/rect.height:1
  };
}

function clusterOverlayItems(items,{point,key,cellPx}){
  const rect=$('map').getBoundingClientRect();
  if(rect.width<=0||rect.height<=0)return [];
  const cellX=Math.max(.1,cellPx*1000/rect.width);
  const cellY=Math.max(.1,cellPx*1000/rect.height);
  const buckets=new Map();
  for(const item of items||[]){
    const p=point(item);
    if(!p||p.x<0||p.x>1000||p.y<0||p.y>1000)continue;
    const itemKey=key(item);
    const bucketKey=itemKey+'|'+Math.floor(p.x/cellX)+'|'+Math.floor(p.y/cellY);
    let bucket=buckets.get(bucketKey);
    if(!bucket){
      bucket={key:itemKey,count:0,sumX:0,sumY:0,sumWorldX:0,sumWorldZ:0,first:item};
      buckets.set(bucketKey,bucket);
    }
    bucket.count++;
    bucket.sumX+=p.x;
    bucket.sumY+=p.y;
    bucket.sumWorldX+=Number(item.x);
    bucket.sumWorldZ+=Number(item.z);
  }
  return [...buckets.values()].map(bucket=>({
    ...bucket,
    x:bucket.sumX/bucket.count,
    y:bucket.sumY/bucket.count,
    worldX:bucket.sumWorldX/bucket.count,
    worldZ:bucket.sumWorldZ/bucket.count
  }));
}

function aggregateOverlayItemsByKey(items,{point,key}){
  return clusterOverlayItemsByKeyPrecision(items,{point,key,perKey:1});
}

function clusterOverlayItemsByKeyPrecision(items,{point,key,perKey=1}){
  const grouped=new Map();
  for(const item of items||[]){
    const p=point(item);
    if(!p||p.x<0||p.x>1000||p.y<0||p.y>1000)continue;
    const itemKey=key(item);
    if(!grouped.has(itemKey))grouped.set(itemKey,[]);
    grouped.get(itemKey).push({item,p});
  }

  const clusters=[];
  const clusterCentroid=entries=>{
    let sumX=0,sumY=0;
    for(const entry of entries){
      sumX+=entry.p.x;
      sumY+=entry.p.y;
    }
    return {x:sumX/entries.length,y:sumY/entries.length};
  };

  const buildCluster=(itemKey,entries)=>{
    const centroid=clusterCentroid(entries);
    let representative=entries[0];
    let bestDistance=Infinity;
    for(const entry of entries){
      const d=(entry.p.x-centroid.x)**2+(entry.p.y-centroid.y)**2;
      if(d<bestDistance){
        representative=entry;
        bestDistance=d;
      }
    }
    return {
      key:itemKey,
      count:entries.length,
      x:representative.p.x,
      y:representative.p.y,
      worldX:Number(representative.item.x),
      worldZ:Number(representative.item.z),
      first:representative.item,
      centroidX:centroid.x,
      centroidY:centroid.y
    };
  };

  for(const [itemKey,entries] of grouped){
    const k=Math.min(Math.max(1,Math.floor(perKey)),entries.length);
    if(k===1){
      clusters.push(buildCluster(itemKey,entries));
      continue;
    }

    const mean=clusterCentroid(entries);
    const centers=[];
    let first=entries[0];
    let firstDistance=Infinity;
    for(const entry of entries){
      const d=(entry.p.x-mean.x)**2+(entry.p.y-mean.y)**2;
      if(d<firstDistance){first=entry;firstDistance=d;}
    }
    centers.push({x:first.p.x,y:first.p.y});

    while(centers.length<k){
      let chosen=entries[0];
      let chosenDistance=-1;
      for(const entry of entries){
        let nearest=Infinity;
        for(const center of centers){
          const d=(entry.p.x-center.x)**2+(entry.p.y-center.y)**2;
          if(d<nearest)nearest=d;
        }
        if(nearest>chosenDistance){chosen=entry;chosenDistance=nearest;}
      }
      centers.push({x:chosen.p.x,y:chosen.p.y});
    }

    let buckets=[];
    for(let iteration=0;iteration<4;iteration++){
      buckets=Array.from({length:k},()=>[]);
      for(const entry of entries){
        let best=0;
        let bestDistance=Infinity;
        for(let i=0;i<centers.length;i++){
          const center=centers[i];
          const d=(entry.p.x-center.x)**2+(entry.p.y-center.y)**2;
          if(d<bestDistance){best=i;bestDistance=d;}
        }
        buckets[best].push(entry);
      }
      for(let i=0;i<k;i++){
        if(!buckets[i].length)continue;
        const bucket=buildCluster(itemKey,buckets[i]);
        centers[i]={x:bucket.centroidX,y:bucket.centroidY};
      }
    }

    for(const bucket of buckets){
      if(bucket.length)clusters.push(buildCluster(itemKey,bucket));
    }
  }
  return clusters;
}

function structureSpatialGroupingKey(candidate){
  if(mapZoom<5)return 'all';
  if(mapZoom<10)return structureMarkerKind(candidate.setId);
  return candidate.setId;
}

function structureSpatialCellPx(){
  if(mapZoom<5)return 210;
  if(mapZoom<10)return 160;
  if(mapZoom<16)return 125;
  if(mapZoom<24)return 92;
  if(mapZoom<32)return 68;
  if(mapZoom<48)return 52;
  return 40;
}

function structureSpatialBudget(){
  if(mapZoom<5)return 18;
  if(mapZoom<10)return 30;
  if(mapZoom<16)return 44;
  if(mapZoom<24)return 64;
  if(mapZoom<32)return 96;
  if(mapZoom<48)return 128;
  return 160;
}

function clusterStructureCandidatesBySpace(items,{cellPx,key}){
  const rect=$('map').getBoundingClientRect();
  if(rect.width<=0||rect.height<=0)return [];
  const cellX=Math.max(.1,cellPx*1000/rect.width);
  const cellY=Math.max(.1,cellPx*1000/rect.height);
  const buckets=new Map();

  for(const item of items||[]){
    const p=worldToOverlay(item.x,item.z);
    if(!p||p.x<0||p.x>1000||p.y<0||p.y>1000)continue;
    const groupKey=key(item);
    const bucketKey=groupKey+'|'+Math.floor(p.x/cellX)+'|'+Math.floor(p.y/cellY);
    if(!buckets.has(bucketKey))buckets.set(bucketKey,{key:groupKey,entries:[]});
    buckets.get(bucketKey).entries.push({item,p});
  }

  const out=[];
  for(const bucket of buckets.values()){
    let sumX=0,sumY=0;
    const typeCounts={};
    for(const entry of bucket.entries){
      sumX+=entry.p.x;
      sumY+=entry.p.y;
      typeCounts[entry.item.setId]=(typeCounts[entry.item.setId]||0)+1;
    }
    const centroidX=sumX/bucket.entries.length;
    const centroidY=sumY/bucket.entries.length;
    let representative=bucket.entries[0];
    let bestDistance=Infinity;
    for(const entry of bucket.entries){
      const d=(entry.p.x-centroidX)**2+(entry.p.y-centroidY)**2;
      if(d<bestDistance){
        representative=entry;
        bestDistance=d;
      }
    }
    out.push({
      key:bucket.key,
      count:bucket.entries.length,
      x:representative.p.x,
      y:representative.p.y,
      worldX:Number(representative.item.x),
      worldZ:Number(representative.item.z),
      first:representative.item,
      typeCount:Object.keys(typeCounts).length,
      typeCounts,
      centroidX,
      centroidY
    });
  }
  return out;
}

function overlayPointVisible(p,padPx=18){
  const img=$('map').getBoundingClientRect();
  const wrap=$('mapWrap').getBoundingClientRect();
  if(img.width<=0||img.height<=0)return false;
  const x=img.left+p.x/1000*img.width;
  const y=img.top+p.y/1000*img.height;
  return x>=wrap.left-padPx&&x<=wrap.right+padPx&&y>=wrap.top-padPx&&y<=wrap.bottom+padPx;
}

function limitStructureClusters(clusters,maxTotal){
  if(clusters.length<=maxTotal)return clusters;
  const groups=new Map();
  for(const cluster of clusters){
    if(!groups.has(cluster.key))groups.set(cluster.key,[]);
    groups.get(cluster.key).push(cluster);
  }
  for(const group of groups.values()){
    group.sort((a,b)=>b.count-a.count);
  }
  const orderedKeys=[...groups.keys()].sort((a,b)=>{
    const ga=groups.get(a),gb=groups.get(b);
    const pa=ga?.[0]?structureCandidatePriority(ga[0].first):0;
    const pb=gb?.[0]?structureCandidatePriority(gb[0].first):0;
    return pb-pa;
  });
  const selected=[];
  let round=0;
  while(selected.length<maxTotal){
    let added=false;
    for(const key of orderedKeys){
      const item=groups.get(key)?.[round];
      if(!item)continue;
      selected.push(item);
      added=true;
      if(selected.length>=maxTotal)break;
    }
    if(!added)break;
    round++;
  }
  return selected;
}

function declutterOverlayClusters(clusters,minGapPx,priority=()=>0){
  const rect=$('map').getBoundingClientRect();
  if(rect.width<=0||rect.height<=0||clusters.length<2)return clusters;
  const ordered=[...clusters].sort((a,b)=>{
    const p=priority(b)-priority(a);
    if(p!==0)return p;
    return b.count-a.count;
  });
  const kept=[];
  for(const cluster of ordered){
    const px=cluster.x/1000*rect.width;
    const py=cluster.y/1000*rect.height;
    let blocked=false;
    for(const other of kept){
      const ox=other.x/1000*rect.width;
      const oy=other.y/1000*rect.height;
      if(Math.hypot(px-ox,py-oy)<minGapPx){
        blocked=true;
        break;
      }
    }
    if(!blocked)kept.push(cluster);
  }
  return kept;
}

function appendClusterMarker(group,cluster,{className,title,targetZoom,radiusScale=1,glyphId='',onClick=null}){
  const ns='http://www.w3.org/2000/svg';
  const scale=overlayUnitScale();
  const radiusPx=(cluster.count>=100?11:cluster.count>=10?10:9)*radiusScale;
  const marker=document.createElementNS(ns,'g');
  marker.setAttribute('class','feature-cluster '+className);
  marker.setAttribute('transform',`translate(${cluster.x} ${cluster.y})`);
  marker.style.pointerEvents='all';
  marker.style.cursor='pointer';

  if(className.includes('structure-stronghold')){
    const ring=document.createElementNS(ns,'ellipse');
    ring.setAttribute('rx',(radiusPx+3)*scale.x);
    ring.setAttribute('ry',(radiusPx+3)*scale.y);
    ring.setAttribute('class','feature-cluster-ring');
    marker.appendChild(ring);
  }

  let halo;
  if(className.includes('structure-settlement')){
    halo=document.createElementNS(ns,'rect');
    halo.setAttribute('x',-radiusPx*scale.x);
    halo.setAttribute('y',-radiusPx*scale.y);
    halo.setAttribute('width',radiusPx*2*scale.x);
    halo.setAttribute('height',radiusPx*2*scale.y);
    halo.setAttribute('rx',3*scale.x);
    halo.setAttribute('ry',3*scale.y);
  }else if(className.includes('structure-underground')){
    halo=document.createElementNS(ns,'polygon');
    halo.setAttribute('points',[
      [0,-radiusPx*scale.y],
      [radiusPx*.86*scale.x,-radiusPx*.5*scale.y],
      [radiusPx*.86*scale.x,radiusPx*.5*scale.y],
      [0,radiusPx*scale.y],
      [-radiusPx*.86*scale.x,radiusPx*.5*scale.y],
      [-radiusPx*.86*scale.x,-radiusPx*.5*scale.y]
    ].map(p=>p.join(',')).join(' '));
  }else if(className.includes('structure-landmark')||className.includes('structure-stronghold')){
    halo=document.createElementNS(ns,'polygon');
    halo.setAttribute('points',[
      [0,-radiusPx*scale.y],
      [radiusPx*scale.x,0],
      [0,radiusPx*scale.y],
      [-radiusPx*scale.x,0]
    ].map(p=>p.join(',')).join(' '));
  }else if(className.includes('structure-portal')){
    halo=document.createElementNS(ns,'polygon');
    halo.setAttribute('points',[
      [0,-radiusPx*scale.y],
      [radiusPx*scale.x,radiusPx*.82*scale.y],
      [-radiusPx*scale.x,radiusPx*.82*scale.y]
    ].map(p=>p.join(',')).join(' '));
  }else{
    halo=document.createElementNS(ns,'ellipse');
    halo.setAttribute('rx',radiusPx*scale.x);
    halo.setAttribute('ry',radiusPx*scale.y);
  }
  halo.setAttribute('class','feature-cluster-bubble');
  marker.appendChild(halo);

  if(glyphId){
    appendStructureGlyph(marker,glyphId,scale.x*.9,scale.y*.9);
  }

  if(cluster.count>1){
    if(glyphId){
      const badge=document.createElementNS(ns,'ellipse');
      badge.setAttribute('cx',String(radiusPx*.72*scale.x));
      badge.setAttribute('cy',String(-radiusPx*.72*scale.y));
      badge.setAttribute('rx',String(6.2*scale.x));
      badge.setAttribute('ry',String(5.2*scale.y));
      badge.setAttribute('class','feature-cluster-badge');
      marker.appendChild(badge);
      const text=document.createElementNS(ns,'text');
      text.setAttribute('x',String(radiusPx*.72*scale.x));
      text.setAttribute('y',String((-radiusPx*.72+1.9)*scale.y));
      text.setAttribute('text-anchor','middle');
      text.setAttribute('class','feature-cluster-badge-count');
      text.setAttribute('font-size',String(7.2*scale.y));
      text.textContent=cluster.approx?'+':(cluster.count>99?'99+':String(cluster.count));
      marker.appendChild(text);
    }else{
      const text=document.createElementNS(ns,'text');
      text.setAttribute('x','0');
      text.setAttribute('y',String(3.2*scale.y));
      text.setAttribute('text-anchor','middle');
      text.setAttribute('class','feature-cluster-count');
      text.setAttribute('font-size',String(9.5*scale.y));
      text.textContent=cluster.approx
        ? '+'
        : (cluster.count>=1000?Math.round(cluster.count/100)/10+'k':String(cluster.count));
      marker.appendChild(text);
    }
  }

  const tooltip=document.createElementNS(ns,'title');
  tooltip.textContent=title(cluster);
  marker.appendChild(tooltip);
  marker.addEventListener('click',e=>{
    e.stopPropagation();
    if(onClick)onClick(cluster);
    centerMapOnWorld(cluster.worldX,cluster.worldZ,targetZoom(cluster));
  });
  group.appendChild(marker);
}

function structureClusterPriority(cluster){
  const kind=cluster.first?structureMarkerKind(cluster.first.setId):cluster.key;
  if(kind==='stronghold')return 100;
  if(kind==='settlement')return 70;
  if(kind==='landmark')return 60;
  if(kind==='ocean')return 50;
  if(kind==='underground')return 40;
  if(kind==='portal')return 30;
  if(kind==='treasure')return 20;
  return 10;
}

function structureCandidatePriority(candidate){
  const id=candidate?.setId||'';
  if(id==='minecraft:strongholds')return 1000;
  if(id==='minecraft:woodland_mansions')return 950;
  if(id==='minecraft:villages')return 900;
  if(id==='minecraft:ocean_monuments')return 860;
  if(id==='minecraft:ancient_cities'||id==='minecraft:trial_chambers')return 820;
  if(id==='minecraft:mineshafts')return 120;
  if(id==='minecraft:buried_treasures')return 80;
  return 500+structureClusterPriority({first:candidate,count:1});
}

function appendStructureSingleMarker(group,candidate){
  const ns='http://www.w3.org/2000/svg';
  const p=worldToOverlay(candidate.x,candidate.z);
  if(p.x<0||p.x>1000||p.y<0||p.y>1000)return false;
  const kind=structureMarkerKind(candidate.setId);
  const scale=overlayUnitScale();
  const radiusPx=8.5;
  const mark=document.createElementNS(ns,'g');
  mark.setAttribute('class','structure-icon structure-'+kind);
  mark.setAttribute('transform',`translate(${p.x} ${p.y})`);
  mark.dataset.x=candidate.x;
  mark.dataset.z=candidate.z;

  const bubble=document.createElementNS(ns,'ellipse');
  bubble.setAttribute('rx',String(radiusPx*scale.x));
  bubble.setAttribute('ry',String(radiusPx*scale.y));
  bubble.setAttribute('class','structure-icon-bg');
  mark.appendChild(bubble);
  appendStructureGlyph(mark,candidate.setId,scale.x*.82,scale.y*.82);

  if(kind==='stronghold'){
    const ring=document.createElementNS(ns,'ellipse');
    ring.setAttribute('rx',String((radiusPx+2.5)*scale.x));
    ring.setAttribute('ry',String((radiusPx+2.5)*scale.y));
    ring.setAttribute('class','structure-icon-ring');
    mark.insertBefore(ring,bubble);
  }

  const title=document.createElementNS(ns,'title');
  title.textContent=
    structureSetName(candidate.setId)+' · 坐标 '+candidate.x+', '+candidate.z+
    ' · 区块 '+candidate.chunkX+', '+candidate.chunkZ+
    (candidate.setId==='minecraft:strongholds'
      ? ' · 精确强要塞位置'
      : ' · 原版结构放置候选位置');
  mark.appendChild(title);
  mark.addEventListener('click',e=>{
    e.stopPropagation();
    showStructureDetail(candidate);
    void focusWorldPoint(candidate.x,candidate.z);
  });
  group.appendChild(mark);
  return true;
}

function mineralClusterZoom(){
  return mapZoom<7?7.25:10.5;
}

function renderFeatureOverlay(){
  const svg=$('featureOverlay');
  if(!svg)return;
  svg.innerHTML='';
  if(!active||!previewFeatures||$('mapWrap').classList.contains('empty')){
    positionFeatureOverlay();
    return;
  }
  const enabled=enabledFeatureKinds();
  const ns='http://www.w3.org/2000/svg';

  if(enabled.has('rivers')){
    const group=document.createElementNS(ns,'g');
    group.setAttribute('class','feature-rivers');
    for(const river of previewFeatures.rivers||[]){
      const a=worldToOverlay(river.x0,river.z0);
      const b=worldToOverlay(river.x1,river.z1);
      if((a.x<0&&b.x<0)||(a.x>1000&&b.x>1000)||(a.y<0&&b.y<0)||(a.y>1000&&b.y>1000))continue;
      const line=document.createElementNS(ns,'line');
      line.setAttribute('x1',a.x); line.setAttribute('y1',a.y);
      line.setAttribute('x2',b.x); line.setAttribute('y2',b.y);
      const worldScale=(active.geometry.SpanXBlocks+active.geometry.SpanZBlocks)/2;
      line.setAttribute('stroke-width',Math.max(.7,river.halfWidth/worldScale*1800));
      line.setAttribute('class',river.oceanMouth?'river-line ocean-mouth':river.lakeMouth?'river-line lake-mouth':'river-line');
      group.appendChild(line);
    }
    svg.appendChild(group);
  }

  if(enabled.has('lakes')){
    const group=document.createElementNS(ns,'g');
    group.setAttribute('class','feature-lakes');
    const worldScale=(active.geometry.SpanXBlocks+active.geometry.SpanZBlocks)/2;
    for(const lake of previewFeatures.lakes||[]){
      const p=worldToOverlay(lake.x,lake.z);
      if(p.x<0||p.x>1000||p.y<0||p.y>1000)continue;
      const circle=document.createElementNS(ns,'circle');
      circle.setAttribute('cx',p.x); circle.setAttribute('cy',p.y);
      circle.setAttribute('r',Math.max(1.2,lake.outerRadius/worldScale*1800));
      circle.setAttribute('class','lake-shape');
      const title=document.createElementNS(ns,'title');
      title.textContent='湖泊 #'+lake.lakeId+' · '+Math.round(lake.x)+', '+Math.round(lake.z);
      circle.appendChild(title);
      group.appendChild(circle);
    }
    svg.appendChild(group);
  }

  if(enabled.has('minerals')&&mineralBodies){
    const group=document.createElementNS(ns,'g');
    group.setAttribute('class','feature-minerals');
    if(mapZoom<FULL_DETAIL_ZOOM-.001){
      const clusters=aggregateOverlayItemsByKey(mineralBodies.bodies||[],{
        point:body=>worldToOverlay(body.x,body.z),
        key:body=>String(body.mineral||'UNKNOWN')
      });
      for(const cluster of clusters){
        cluster.approx=Boolean(mineralBodies.truncated);
        const mineral=String(cluster.key||'unknown').toLowerCase();
        appendClusterMarker(group,cluster,{
          className:'mineral-cluster mineral-'+mineral,
          title:c=>mineralName(c.key)+' · 当前视野 '+c.count+' 个矿体 · 最大缩放后全部展开',
          targetZoom:()=>Math.min(MAX_MAP_ZOOM,Math.max(mapZoom*1.8,mapZoom+2))
        });
      }
    }else{
      const worldScale=(active.geometry.SpanXBlocks+active.geometry.SpanZBlocks)/2;
      for(const body of mineralBodies.bodies||[]){
        const a=worldToOverlay(body.startX,body.startZ);
        const m=worldToOverlay(body.midX,body.midZ);
        const b=worldToOverlay(body.endX,body.endZ);
        const path=document.createElementNS(ns,'path');
        path.setAttribute('d',`M ${a.x} ${a.y} Q ${m.x} ${m.y} ${b.x} ${b.y}`);
        const mineral=String(body.mineral||'').toLowerCase();
        const source=String(body.source||'').toLowerCase();
        path.setAttribute('class',`mineral-body mineral-${mineral} source-${source}${body.large?' major':''}`);
        path.setAttribute('stroke-width',Math.max(1.2/mapZoom,body.radius/worldScale*2000));
        path.dataset.x=body.x;
        path.dataset.z=body.z;
        const title=document.createElementNS(ns,'title');
        title.textContent=
          mineralName(body.mineral)+' · '+mineralSourceName(body.source)+
          ' · '+Math.round(body.x)+', Y '+Math.round(body.y)+', '+Math.round(body.z);
        path.appendChild(title);
        path.addEventListener('click',e=>{
          e.stopPropagation();
          showMineralDetail(body);
          void focusWorldPoint(body.x,body.z);
        });
        group.appendChild(path);
      }
    }
    svg.appendChild(group);
  }

  if(enabled.has('structures')&&structureCandidates){
    const group=document.createElementNS(ns,'g');
    group.setAttribute('class','feature-structures');
    const visibleCandidates=(structureCandidates.candidates||[]).filter(candidate=>
      overlayPointVisible(worldToOverlay(candidate.x,candidate.z),24)
    );
    if(mapZoom<FULL_DETAIL_ZOOM-.001){
      let clusters=clusterStructureCandidatesBySpace(visibleCandidates,{
        cellPx:structureSpatialCellPx(),
        key:structureSpatialGroupingKey
      });
      clusters=limitStructureClusters(clusters,structureSpatialBudget())
        .sort((a,b)=>structureCandidatePriority(b.first)-structureCandidatePriority(a.first));
      for(const cluster of clusters){
        cluster.approx=Boolean(structureCandidates.truncated);
        const markerKind=cluster.key==='all'
          ? 'mixed'
          : (mapZoom<10?cluster.key:structureMarkerKind(cluster.first.setId));
        appendClusterMarker(group,cluster,{
          className:'structure-cluster structure-'+markerKind,
          title:c=>c.typeCount>1
            ? '该区域约 '+c.count+' 个候选 · '+c.typeCount+' 种结构 · 放大后继续拆分'
            : structureSetName(c.first.setId)+' · 该区域约 '+c.count+' 个候选 · 放大后继续拆分',
          targetZoom:()=>Math.min(MAX_MAP_ZOOM,Math.max(mapZoom*1.65,mapZoom+1.5)),
          radiusScale:markerKind==='stronghold'?1.05:1,
          glyphId:cluster.typeCount>1?'geoworld:mixed_structures':cluster.first.setId,
          onClick:showStructureClusterDetail
        });
      }
    }else{
      for(const candidate of visibleCandidates){
        appendStructureSingleMarker(group,candidate);
      }
    }
    svg.appendChild(group);
  }

  if(enabled.has('peaks')){
    const group=document.createElementNS(ns,'g');
    group.setAttribute('class','feature-peaks');
    if(mapZoom<FULL_DETAIL_ZOOM-.001){
      const bounds=mineralViewportBounds();
      const visiblePeaks=(previewFeatures.peaks||[]).filter(peak=>
        !bounds||(peak.x>=bounds.minX&&peak.x<=bounds.maxX&&peak.z>=bounds.minZ&&peak.z<=bounds.maxZ)
      );
      const clusters=aggregateOverlayItemsByKey(visiblePeaks,{
        point:peak=>worldToOverlay(peak.x,peak.z),
        key:()=> 'peak'
      });
      for(const cluster of clusters){
        appendClusterMarker(group,cluster,{
          className:'peak-cluster',
          title:c=>'山峰 · 当前视野 '+c.count+' 个 · 最大缩放后全部展开',
          targetZoom:()=>Math.min(MAX_MAP_ZOOM,Math.max(mapZoom*1.8,mapZoom+2)),
          radiusScale:.9
        });
      }
    }else{
      const markerScale=overlayUnitScale();
      for(const peak of previewFeatures.peaks||[]){
        const p=worldToOverlay(peak.x,peak.z);
        if(p.x<0||p.x>1000||p.y<0||p.y>1000)continue;
        const sx=4*markerScale.x;
        const sy=4*markerScale.y;
        const mark=document.createElementNS(ns,'path');
        mark.setAttribute('d',`M ${p.x} ${p.y-sy} L ${p.x-sx} ${p.y+sy*.75} L ${p.x+sx} ${p.y+sy*.75} Z`);
        mark.setAttribute('class','peak-mark');
        mark.dataset.x=peak.x; mark.dataset.z=peak.z;
        const title=document.createElementNS(ns,'title');
        title.textContent=(peak.name||'山峰')+' · '+Math.round(peak.x)+', '+Math.round(peak.z)+' · '+(peak.detail||'');
        mark.appendChild(title);
        mark.addEventListener('click',e=>{e.stopPropagation();void focusWorldPoint(peak.x,peak.z)});
        group.appendChild(mark);
      }
    }
    svg.appendChild(group);
  }
  positionFeatureOverlay();
}

function positionFeatureOverlay(){
  const svg=$('featureOverlay');
  const img=$('map');
  if(!svg||!img)return;
  const imgRect=img.getBoundingClientRect();
  const wrapRect=$('mapWrap').getBoundingClientRect();
  if(imgRect.width<=0||imgRect.height<=0){
    svg.style.display='none';
    return;
  }
  svg.style.display='block';
  svg.style.left=(imgRect.left-wrapRect.left)+'px';
  svg.style.top=(imgRect.top-wrapRect.top)+'px';
  svg.style.width=imgRect.width+'px';
  svg.style.height=imgRect.height+'px';
}

function setMobileSheet(which,open){
  const controls=document.querySelector('.controls');
  const inspector=document.querySelector('.inspector');
  if(which==='settings'){
    controls.classList.toggle('mobile-open',open);
    if(open)inspector.classList.remove('mobile-open');
  }else{
    inspector.classList.toggle('mobile-open',open);
    if(open)controls.classList.remove('mobile-open');
  }
  document.body.classList.toggle(
    'mobile-sheet-visible',
    controls.classList.contains('mobile-open')||inspector.classList.contains('mobile-open')
  );
}
$('mobileOpenSettings').onclick=()=>setMobileSheet('settings',true);
$('mobileOpenFeatures').onclick=()=>setMobileSheet('features',true);
$('mobileCloseSettings').onclick=()=>setMobileSheet('settings',false);
$('mobileCloseFeatures').onclick=()=>setMobileSheet('features',false);

document.querySelectorAll('.nav-item,.route-button').forEach(btn=>btn.addEventListener('click',()=>navigatePage(btn.dataset.page)));
document.querySelectorAll('input[name="acceptanceType"]').forEach(radio=>radio.addEventListener('change',()=>{
  acceptanceData=null;
  clearAcceptanceView();
  renderAcceptanceSummary();
  if(active)void generateAcceptanceTable();
}));
$('acceptanceGenerate').onclick=generateAcceptanceTable;
$('acceptanceBigFilter').onchange=renderAcceptanceTable;
document.querySelectorAll('.acceptance-download').forEach(btn=>btn.onclick=()=>downloadAcceptance(btn.dataset.format));
document.querySelectorAll('.feature-toggle').forEach(input=>input.onchange=()=>{
  if(input.dataset.feature==='minerals'){
    if(input.checked){
      scheduleMineralRefresh(0);
    }else{
      mineralBodies=null;
      mineralLoadToken++;
      $('mineralStatus').textContent='真实规划矿体已隐藏。';
      $('mineralDetail').hidden=true;
    }
  }
  if(input.dataset.feature==='structures'){
    if(input.checked){
      scheduleStructureRefresh(0);
    }else{
      structureCandidates=null;
      structureLoadToken++;
      $('structureStatus').textContent='原版结构候选已隐藏。';
      $('structureDetail').hidden=true;
    }
  }
  renderFeatureOverlay();
});

$('map').addEventListener('load',()=>{
  renderExportOverlay();
  renderCursorOverlay();
  renderFeatureOverlay();
  updateActionState();
});
window.addEventListener('resize',()=>{
  renderExportOverlay();
  renderCursorOverlay();
  renderFeatureOverlay();
});
window.addEventListener('beforeunload',()=>{if(mapObjectURL)URL.revokeObjectURL(mapObjectURL)});

updateRangePreview();
updateContinentLabels();
syncContinentPreset();
updateWorldProfileControls();
updateLayerHelp();
updateActionState();
clearAcceptanceView();
renderHomeCurrentWorld();
const initialPage=['home','world','acceptance','models','params','about'].includes(location.hash.slice(1))?location.hash.slice(1):'world';
navigatePage(initialPage);
refreshModels();
refreshStatus();
setInterval(()=>{if(!poll)void refreshStatus()},2000);
