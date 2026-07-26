.. title:: Kubeflow Pipelines

.. meta::
   :description: Kubeflow Pipelines — build and deploy portable, scalable machine learning workflows on Kubernetes
   :keywords: kubeflow, pipelines, kfp, machine learning, workflows, kubernetes

.. raw:: html

   <section class="kfp-hero">
   <div class="kfp-hero-inner">

     <div class="kfp-hero-badge-row">
       <div class="kfp-hero-badge">Open Source <span class="kfp-dot"></span> CNCF <span class="kfp-dot"></span> Kubeflow</div>
     </div>

     <div class="kfp-hero-grid">

       <div class="kfp-hero-text">
         <h1 class="kfp-hero-title">Kubeflow Pipelines</h1>
         <div class="kfp-hero-rule"></div>
         <p class="kfp-hero-tagline">
           Build and deploy portable, scalable machine learning workflows
           using containers on Kubernetes.
         </p>
         <div class="kfp-hero-actions">
           <a href="overview.html" class="kfp-btn kfp-btn-primary">Get Started
           <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><path d="M5 12h14M12 5l7 7-7 7"/></svg></a>
           <a href="https://github.com/kubeflow/pipelines" class="kfp-btn kfp-btn-secondary">GitHub
           <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M12 .5a12 12 0 0 0-3.8 23.4c.6.1.8-.3.8-.6v-2c-3.3.7-4-1.6-4-1.6-.6-1.4-1.4-1.8-1.4-1.8-1-.7.1-.7.1-.7 1.2.1 1.8 1.2 1.8 1.2 1 1.8 2.8 1.3 3.5 1 .1-.8.4-1.3.7-1.6-2.7-.3-5.5-1.3-5.5-5.9 0-1.3.5-2.4 1.2-3.2-.1-.3-.5-1.5.1-3.2 0 0 1-.3 3.3 1.2a11.5 11.5 0 0 1 6 0C17.3 4.9 18.3 5.2 18.3 5.2c.6 1.7.2 2.9.1 3.2.8.8 1.2 1.9 1.2 3.2 0 4.6-2.8 5.6-5.5 5.9.4.4.8 1.1.8 2.2v3.3c0 .3.2.7.8.6A12 12 0 0 0 12 .5z"/></svg></a>
         </div>
         <div class="kfp-hero-sub">
           Author pipelines in Python with the KFP SDK, then run them on any
           KFP-conformant backend.
         </div>
       </div>

       <div class="kfp-hero-visual" aria-hidden="true">
         <div class="kfp-logo-stage">
           <div class="kfp-logo-aura"></div>
           <img class="kfp-hero-logo" src="_images/pipelines-icon.svg" alt="" />
         </div>
       </div>

     </div>
   </div>
   </section>

   <section class="kfp-what">
     <div class="kfp-section-inner">
       <h2 class="kfp-section-title">What is Kubeflow Pipelines?</h2>
       <p class="kfp-section-lead">
         Kubeflow Pipelines (KFP) is a platform for building and deploying
         portable, scalable machine learning workflows using containers on
         Kubernetes. Author pipelines in Python with the KFP SDK, compile them
         to a platform-neutral <code>IR YAML</code>, and run them unchanged on
         any KFP-conformant backend, such as the open source KFP backend or
         Google Cloud Vertex AI Pipelines.
       </p>
       <p class="kfp-section-lead">
         Kubeflow Pipelines is the workflow orchestration component of
         <a href="https://www.kubeflow.org/">Kubeflow</a>, the open-source ML
         platform for Kubernetes.
       </p>
     </div>
   </section>

   <section class="kfp-why">
     <div class="kfp-section-inner">
       <h2 class="kfp-section-title">Why Kubeflow Pipelines?</h2>
       <div class="kfp-feature-grid">

         <div class="kfp-feature-card">
           <div class="kfp-feature-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M16 18l6-6-6-6M8 6l-6 6 6 6"/></svg></div>
           <h3>Pythonic Authoring</h3>
           <p>Turn a Python function into a pipeline step with a single decorator. Author components and pipelines with the <code>kfp</code> SDK &mdash; no YAML by hand.</p>
         </div>

         <div class="kfp-feature-card">
           <div class="kfp-feature-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><path d="M3.27 6.96L12 12.01l8.73-5.05M12 22.08V12"/></svg></div>
           <h3>Portable &amp; Reproducible</h3>
           <p>Compile pipelines to a platform-neutral IR YAML that runs unchanged across environments, so results are consistent and easy to reproduce.</p>
         </div>

         <div class="kfp-feature-card">
           <div class="kfp-feature-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/></svg></div>
           <h3>Reusable Components</h3>
           <p>Build once, share everywhere. Compose pipelines from your own reusable components and an ecosystem of existing ones.</p>
         </div>

         <div class="kfp-feature-card">
           <div class="kfp-feature-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 3v18h18"/><path d="M7 14l4-4 3 3 5-6"/></svg></div>
           <h3>Track Everything</h3>
           <p>Automatically track parameters, artifacts, runs, and experiments with built-in ML metadata and end-to-end lineage.</p>
         </div>

         <div class="kfp-feature-card">
           <div class="kfp-feature-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/></svg></div>
           <h3>Efficient Execution</h3>
           <p>Run tasks in parallel and skip redundant work with caching, so pipelines finish faster and use fewer resources.</p>
         </div>

         <div class="kfp-feature-card">
           <div class="kfp-feature-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><path d="M8.6 13.5l6.8 4M15.4 6.5l-6.8 4"/></svg></div>
           <h3>Pass Data Between Steps</h3>
           <p>Easily pass parameters and ML artifacts between components, so each step consumes the outputs of the steps before it.</p>
         </div>

       </div>
     </div>
   </section>

   <section class="kfp-concepts">
     <div class="kfp-section-inner">
       <h2 class="kfp-section-title">Core Concepts</h2>
       <div class="kfp-chip-row">
         <a class="kfp-chip" href="concepts/pipeline.html">Pipeline</a>
         <a class="kfp-chip" href="concepts/component.html">Component</a>
         <a class="kfp-chip" href="concepts/step.html">Step</a>
         <a class="kfp-chip" href="concepts/graph.html">Graph</a>
         <a class="kfp-chip" href="concepts/output-artifact.html">Artifact</a>
         <a class="kfp-chip" href="concepts/experiment.html">Experiment</a>
         <a class="kfp-chip" href="concepts/run.html">Run</a>
         <a class="kfp-chip" href="concepts/ir-yaml.html">IR YAML</a>
         <a class="kfp-chip" href="concepts/metadata.html">ML Metadata</a>
       </div>
     </div>
   </section>

   <section class="kfp-how">
     <div class="kfp-section-inner">
       <h2 class="kfp-section-title">How It Works</h2>
       <p class="kfp-section-lead">
         You author a pipeline in Python and compile it to a platform-neutral
         <code>IR YAML</code>, then submit it with the SDK, CLI, or UI. The KFP
         API Server records the run and Argo orchestrates it, running each step
         as a container while artifacts, metadata, and status stream back to the
         Pipeline UI.
       </p>
       <div class="kfp-how-diagram-wrap">
         <img class="kfp-how-diagram kfp-how-diagram-dark" src="_images/pipelines-how-it-works.png"
              alt="How a Kubeflow Pipeline goes from Python to running pods. 1. Author: write a pipeline in Python with the KFP SDK. 2. Compile: the SDK compiles it to a platform-neutral IR YAML. 3. Submit: send it to the KFP API Server via the SDK, CLI, or UI. 4. Orchestrate: the Argo Workflow Controller runs the pipeline. 5. Track: a driver pod coordinates the run and executor pods run each step as a container, reporting status, artifacts, and metadata to ML Metadata and the object store, all viewable in the Pipeline UI." />
         <img class="kfp-how-diagram kfp-how-diagram-light" src="_images/pipelines-how-it-works-light.png" alt="" />
       </div>
     </div>
   </section>

   <section class="kfp-docs">
     <div class="kfp-section-inner">
       <h2 class="kfp-section-title kfp-docs-title">Documentation</h2>
       <p class="kfp-section-lead">
         Everything you need, from your first pipeline to running in
         production and contributing back.
       </p>
     </div>
   </section>

.. grid:: 1 2 2 3
   :gutter: 3
   :class-container: kfp-landing-grid

   .. grid-item-card:: Overview
      :link: overview
      :link-type: doc

      Learn what Kubeflow Pipelines is, who it is for, and why you would use it

   .. grid-item-card:: Getting Started
      :link: getting-started
      :link-type: doc

      Install KFP and run your first pipeline in minutes

   .. grid-item-card:: Interfaces
      :link: interfaces
      :link-type: doc

      Author, run, and manage pipelines with the SDK, CLI, and web UI

   .. grid-item-card:: Concepts
      :link: concepts/index
      :link-type: doc

      Understand pipelines, components, graphs, runs, experiments, and artifacts

   .. grid-item-card:: User Guides
      :link: user-guides/index
      :link-type: doc

      Author components and pipelines, pass data between tasks, and control flow

   .. grid-item-card:: Operator Guides
      :link: operator-guides/index
      :link-type: doc

      Install, configure, and operate Kubeflow Pipelines on a cluster

   .. grid-item-card:: Python SDK
      :link: python-sdk
      :link-type: doc

      Install the ``kfp`` SDK, then look up the API and CLI reference

   .. grid-item-card:: Reference
      :link: reference/index
      :link-type: doc

      Look up API specifications, version compatibility, and component definitions

   .. grid-item-card:: Contributor Guide
      :link: contributing/index
      :link-type: doc

      Set up your development environment and contribute to Kubeflow Pipelines

.. raw:: html

   <section class="kfp-quickstart">
     <div class="kfp-section-inner">
       <h2 class="kfp-section-title">Build a Pipeline in Python</h2>
       <div class="kfp-code-window">
         <div class="kfp-code-header">
           <span class="kfp-code-dot"></span>
           <span class="kfp-code-dot"></span>
           <span class="kfp-code-dot"></span>
           <span class="kfp-code-lang">PYTHON</span>
         </div>

.. code-block:: python

   from kfp import dsl

   @dsl.component
   def say_hello(name: str) -> str:
       hello_text = f'Hello, {name}!'
       print(hello_text)
       return hello_text

   @dsl.pipeline
   def hello_pipeline(recipient: str) -> str:
       hello_task = say_hello(name=recipient)
       return hello_task.output

.. raw:: html

       </div>
       <p class="kfp-code-caption">
         Compile it to <code>IR YAML</code>, then run it locally or submit to
         any KFP-conformant backend.
         <a href="getting-started.html">See the full quickstart &rarr;</a>
       </p>
     </div>
   </section>

   <section class="kfp-community">
     <div class="kfp-section-inner">
       <h2 class="kfp-section-title">Join the Community</h2>
       <p class="kfp-section-lead">
         Kubeflow Pipelines is built by an open, welcoming community of
         developers, data scientists, and organizations. Kubeflow is a Cloud
         Native Computing Foundation (CNCF) project.
       </p>
       <div class="kfp-community-grid">

         <a class="kfp-community-card" href="https://github.com/kubeflow/pipelines">
           <div class="kfp-community-icon kfp-ci-github"><svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M12 .5a12 12 0 0 0-3.8 23.4c.6.1.8-.3.8-.6v-2c-3.3.7-4-1.6-4-1.6-.6-1.4-1.4-1.8-1.4-1.8-1-.7.1-.7.1-.7 1.2.1 1.8 1.2 1.8 1.2 1 1.8 2.8 1.3 3.5 1 .1-.8.4-1.3.7-1.6-2.7-.3-5.5-1.3-5.5-5.9 0-1.3.5-2.4 1.2-3.2-.1-.3-.5-1.5.1-3.2 0 0 1-.3 3.3 1.2a11.5 11.5 0 0 1 6 0C17.3 4.9 18.3 5.2 18.3 5.2c.6 1.7.2 2.9.1 3.2.8.8 1.2 1.9 1.2 3.2 0 4.6-2.8 5.6-5.5 5.9.4.4.8 1.1.8 2.2v3.3c0 .3.2.7.8.6A12 12 0 0 0 12 .5z"/></svg></div>
           <h3>GitHub</h3>
           <p>Star, fork, and contribute</p>
         </a>

         <a class="kfp-community-card" href="https://kubeflow.slack.com/channels/kubeflow-pipelines">
           <div class="kfp-community-icon kfp-ci-slack"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg></div>
           <h3>Slack</h3>
           <p>#kubeflow-pipelines</p>
         </a>

         <a class="kfp-community-card" href="https://stackoverflow.com/questions/tagged/kubeflow-pipelines">
           <div class="kfp-community-icon kfp-ci-stack"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 2 7 12 12 22 7 12 2"/><polyline points="2 17 12 22 22 17"/><polyline points="2 12 12 17 22 12"/></svg></div>
           <h3>Stack Overflow</h3>
           <p>Ask and answer questions</p>
         </a>

         <a class="kfp-community-card" href="https://groups.google.com/g/kubeflow-discuss">
           <div class="kfp-community-icon kfp-ci-mail"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="4" width="20" height="16" rx="2"/><path d="m22 7-10 5L2 7"/></svg></div>
           <h3>Mailing List</h3>
           <p>kubeflow-discuss</p>
         </a>

         <a class="kfp-community-card" href="https://github.com/kubeflow/pipelines/issues">
           <div class="kfp-community-icon kfp-ci-issues"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg></div>
           <h3>Issues</h3>
           <p>Report bugs and request features</p>
         </a>

         <a class="kfp-community-card" href="https://www.kubeflow.org/">
           <div class="kfp-community-icon kfp-ci-docs"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></svg></div>
           <h3>Kubeflow.org</h3>
           <p>Official Kubeflow documentation</p>
         </a>

         <a class="kfp-community-card" href="contributing/index.html">
           <div class="kfp-community-icon kfp-ci-contrib"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg></div>
           <h3>Contributing</h3>
           <p>How to get involved</p>
         </a>

         <a class="kfp-community-card" href="https://github.com/kubeflow/pipelines/releases">
           <div class="kfp-community-icon kfp-ci-releases"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z"/><line x1="7" y1="7" x2="7.01" y2="7"/></svg></div>
           <h3>Releases</h3>
           <p>Changelog and downloads</p>
         </a>

       </div>
     </div>
   </section>

   <section class="kfp-footer">
     <div class="kfp-section-inner">
       <p class="kfp-footer-lead">
         Kubeflow Pipelines is part of
         <a href="https://www.kubeflow.org/">Kubeflow</a>, the open-source ML
         platform for Kubernetes.
       </p>
       <a class="kfp-footer-cncf-logo" href="https://www.cncf.io/" aria-label="Cloud Native Computing Foundation">
         <img src="_images/cncf-white.svg" alt="Cloud Native Computing Foundation" />
       </a>
       <p class="kfp-footer-cncf-line">
         Kubeflow is a
         <a href="https://www.cncf.io/">Cloud Native Computing Foundation</a>
         project.
       </p>
       <p class="kfp-footer-copyright">
         &copy; The Kubeflow Authors &middot; Documentation distributed under
         <a href="https://creativecommons.org/licenses/by/4.0/">CC&nbsp;BY&nbsp;4.0</a>
       </p>
     </div>
   </section>

.. only:: html

   .. The hero and sections above are raw HTML, which Sphinx does not scan for
      assets. These hidden directives copy their images into _images/.
   .. image:: images/pipelines-icon.svg
      :width: 0
      :class: hidden
   .. image:: images/pipelines-how-it-works.png
      :width: 0
      :class: hidden
   .. image:: images/pipelines-how-it-works-light.png
      :width: 0
      :class: hidden
   .. image:: images/cncf-white.svg
      :width: 0
      :class: hidden

.. toctree::
   :hidden:

   Overview <overview>
   Getting Started <getting-started>
   interfaces
   concepts/index
   user-guides/index
   operator-guides/index
   Python SDK <python-sdk>
   reference/index
   Contributor Guide <contributing/index>
