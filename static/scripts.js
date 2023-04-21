var coll = document.getElementsByClassName("collapsible");
var i;
for (i = 0; i < coll.length; i++) {
    coll[i].addEventListener("click", function() {
        this.classList.toggle("active");
        var content = this.nextElementSibling;
        if (content.style.display === "block") {
            content.style.display = "none";
        } else {
            content.style.display = "block";
        }
    });
}
function deleteSite(rootDomain, liId) {
    if (!confirm('Are you sure you want to delete this site and its related files?')) {
      return;
    }
  
    fetch('/delete-site?rootDomain=' + encodeURIComponent(rootDomain), {
      method: 'POST'
    })
    .then(response => {
      if (response.ok) {
        // Remove the list item from the page
        const listItem = document.getElementById(liId);
        if (listItem) {
          listItem.remove();
        }
      } else {
        alert('Error deleting site. Please try again.');
      }
    })
    .catch(error => {
      alert('Error deleting site. Please try again.');
    });
  }
  
function filterList() {
    const input = document.getElementById('searchInput');
    const filter = input.value.toUpperCase();
    const ul = document.getElementById('rootDomainList');
    const li = ul.getElementsByTagName('li');
  
    for (let i = 0; i < li.length; i++) {
      const a = li[i].getElementsByTagName('a')[0];
      const txtValue = a.textContent || a.innerText;
      if (txtValue.toUpperCase().indexOf(filter) > -1) {
        li[i].style.display = '';
      } else {
        li[i].style.display = 'none';
      }
    }
  }