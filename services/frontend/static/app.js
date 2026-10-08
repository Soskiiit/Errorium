const root=document.documentElement;
const openButton=document.querySelector('[data-open-menu]');
const closeButtons=document.querySelectorAll('[data-close-menu]');
const query=document.querySelector('#query');
const sidebar=document.querySelector('#sidebar');
const desktopQuery=window.matchMedia('(min-width: 800px)');
function setMenu(open){
  const desktop=desktopQuery.matches;
  root.classList.toggle('sidebar-open',open&&!desktop);
  openButton?.setAttribute('aria-expanded',String(desktop||open));
  sidebar?.setAttribute('aria-hidden',String(!desktop&&!open));
}
openButton?.addEventListener('click',()=>{setMenu(true);sidebar?.querySelector('input,button,a')?.focus();});
closeButtons.forEach(button=>button.addEventListener('click',()=>setMenu(false)));
document.addEventListener('keydown',event=>{if(event.key==='Escape')setMenu(false);});
desktopQuery.addEventListener('change',()=>setMenu(false));
query?.addEventListener('input',()=>{query.style.height='auto';query.style.height=`${Math.min(query.scrollHeight,52)}px`;});
setMenu(false);
