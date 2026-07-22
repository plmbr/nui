---
layout: page
title: Blog
subtitle: Release notes and the occasional deep dive.
permalink: /blog/
---

<ul class="post-list">
  {% for post in site.posts %}
    <li class="post-list__item">
      <p class="post-list__date"><time datetime="{{ post.date | date_to_xmlschema }}">{{ post.date | date: "%B %-d, %Y" }}</time></p>
      <h2 class="post-list__title"><a href="{{ post.url | relative_url }}">{{ post.title | escape }}</a></h2>
      {% if post.description %}
        <p class="post-list__excerpt">{{ post.description | escape }}</p>
      {% else %}
        <p class="post-list__excerpt">{{ post.excerpt | strip_html | truncatewords: 32 }}</p>
      {% endif %}
    </li>
  {% endfor %}
</ul>
