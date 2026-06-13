<template>
  <div class="card floating chmod-modal">
    <div class="card-title">
      <h2>{{ $t("prompts.chmod") }}</h2>
    </div>

    <div class="card-content">
      <p class="break-word">
        <strong>{{ targetName }}</strong>
      </p>

      <!-- File permission -->
      <div class="chmod-section">
        <h3 v-if="recursive">{{ $t("prompts.chmodFileMode") }}</h3>

        <div class="chmod-octal">
          <label>{{ $t("prompts.chmodOctal") }}:</label>
          <input
            type="text"
            v-model="octalInput"
            maxlength="4"
            placeholder="0644"
            @input="onOctalChange"
            class="input input--block"
          />
          <span class="chmod-symbolic">{{ symbolicString }}</span>
        </div>

        <table class="chmod-table">
          <thead>
            <tr>
              <th></th>
              <th>{{ $t("prompts.chmodRead") }}</th>
              <th>{{ $t("prompts.chmodWrite") }}</th>
              <th>{{ $t("prompts.chmodExecute") }}</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td><strong>{{ $t("prompts.chmodOwner") }}</strong></td>
              <td><input type="checkbox" v-model="perms.ownerRead" @change="onCheckboxChange" /></td>
              <td><input type="checkbox" v-model="perms.ownerWrite" @change="onCheckboxChange" /></td>
              <td><input type="checkbox" v-model="perms.ownerExec" @change="onCheckboxChange" /></td>
            </tr>
            <tr>
              <td><strong>{{ $t("prompts.chmodGroup") }}</strong></td>
              <td><input type="checkbox" v-model="perms.groupRead" @change="onCheckboxChange" /></td>
              <td><input type="checkbox" v-model="perms.groupWrite" @change="onCheckboxChange" /></td>
              <td><input type="checkbox" v-model="perms.groupExec" @change="onCheckboxChange" /></td>
            </tr>
            <tr>
              <td><strong>{{ $t("prompts.chmodOthers") }}</strong></td>
              <td><input type="checkbox" v-model="perms.othersRead" @change="onCheckboxChange" /></td>
              <td><input type="checkbox" v-model="perms.othersWrite" @change="onCheckboxChange" /></td>
              <td><input type="checkbox" v-model="perms.othersExec" @change="onCheckboxChange" /></td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Recursive option -->
      <div v-if="isDir" class="chmod-recursive">
        <label>
          <input type="checkbox" v-model="recursive" />
          {{ $t("prompts.chmodRecursive") }}
        </label>
      </div>

      <!-- Directory permission (shown only when recursive) -->
      <div v-if="recursive" class="chmod-section chmod-dir-section">
        <h3>{{ $t("prompts.chmodDirMode") }}</h3>

        <div class="chmod-octal">
          <label>{{ $t("prompts.chmodOctal") }}:</label>
          <input
            type="text"
            v-model="dirOctalInput"
            maxlength="4"
            placeholder="0755"
            @input="onDirOctalChange"
            class="input input--block"
          />
          <span class="chmod-symbolic">{{ dirSymbolicString }}</span>
        </div>

        <table class="chmod-table">
          <thead>
            <tr>
              <th></th>
              <th>{{ $t("prompts.chmodRead") }}</th>
              <th>{{ $t("prompts.chmodWrite") }}</th>
              <th>{{ $t("prompts.chmodExecute") }}</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td><strong>{{ $t("prompts.chmodOwner") }}</strong></td>
              <td><input type="checkbox" v-model="dirPerms.ownerRead" @change="onDirCheckboxChange" /></td>
              <td><input type="checkbox" v-model="dirPerms.ownerWrite" @change="onDirCheckboxChange" /></td>
              <td><input type="checkbox" v-model="dirPerms.ownerExec" @change="onDirCheckboxChange" /></td>
            </tr>
            <tr>
              <td><strong>{{ $t("prompts.chmodGroup") }}</strong></td>
              <td><input type="checkbox" v-model="dirPerms.groupRead" @change="onDirCheckboxChange" /></td>
              <td><input type="checkbox" v-model="dirPerms.groupWrite" @change="onDirCheckboxChange" /></td>
              <td><input type="checkbox" v-model="dirPerms.groupExec" @change="onDirCheckboxChange" /></td>
            </tr>
            <tr>
              <td><strong>{{ $t("prompts.chmodOthers") }}</strong></td>
              <td><input type="checkbox" v-model="dirPerms.othersRead" @change="onDirCheckboxChange" /></td>
              <td><input type="checkbox" v-model="dirPerms.othersWrite" @change="onDirCheckboxChange" /></td>
              <td><input type="checkbox" v-model="dirPerms.othersExec" @change="onDirCheckboxChange" /></td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div class="card-action">
      <button
        @click="closeHovers"
        class="button button--flat button--grey"
        :aria-label="$t('buttons.cancel')"
        :title="$t('buttons.cancel')"
        tabindex="2"
      >
        {{ $t("buttons.cancel") }}
      </button>
      <button
        id="focus-prompt"
        @click="submit"
        class="button button--flat"
        :aria-label="$t('buttons.apply')"
        :title="$t('buttons.apply')"
        tabindex="1"
      >
        {{ $t("buttons.apply") }}
      </button>
    </div>
  </div>
</template>

<script>
import { mapActions, mapState, mapWritableState } from "pinia";
import { files as api } from "@/api";
import buttons from "@/utils/buttons";
import { useFileStore } from "@/stores/file";
import { useLayoutStore } from "@/stores/layout";

export default {
  name: "chmod",
  inject: ["$showError"],
  data() {
    return {
      octalInput: "0644",
      dirOctalInput: "0755",
      recursive: false,
      perms: {
        ownerRead: true,
        ownerWrite: true,
        ownerExec: false,
        groupRead: true,
        groupWrite: false,
        groupExec: false,
        othersRead: true,
        othersWrite: false,
        othersExec: false,
      },
      dirPerms: {
        ownerRead: true,
        ownerWrite: true,
        ownerExec: true,
        groupRead: true,
        groupWrite: false,
        groupExec: true,
        othersRead: true,
        othersWrite: false,
        othersExec: true,
      },
    };
  },
  computed: {
    ...mapState(useFileStore, ["req", "selected", "selectedCount", "isListing"]),
    ...mapState(useLayoutStore, ["currentPrompt"]),
    ...mapWritableState(useFileStore, ["reload"]),
    targetName() {
      if (this.selectedCount === 0 || !this.isListing) {
        return this.req?.name || "";
      }
      if (this.selectedCount === 1) {
        return this.req.items[this.selected[0]]?.name || "";
      }
      return `${this.selectedCount} items`;
    },
    isDir() {
      if (this.selectedCount === 0 || !this.isListing) {
        return this.req?.isDir || false;
      }
      if (this.selectedCount === 1) {
        return this.req.items[this.selected[0]]?.isDir || false;
      }
      return true;
    },
    symbolicString() {
      return this.buildSymbolic(this.perms);
    },
    dirSymbolicString() {
      return this.buildSymbolic(this.dirPerms);
    },
  },
  mounted() {
    let currentMode = null;
    if (this.selectedCount === 1 && this.isListing) {
      currentMode = this.req.items[this.selected[0]]?.mode;
    } else if (this.selectedCount === 0 && this.req) {
      currentMode = this.req.mode;
    }
    if (currentMode != null) {
      const permBits = currentMode & 0o777;
      this.octalInput = "0" + permBits.toString(8).padStart(3, "0");
      this.onOctalChange();
    }
  },
  methods: {
    ...mapActions(useLayoutStore, ["closeHovers"]),
    buildSymbolic(p) {
      const r = (v) => (v ? "r" : "-");
      const w = (v) => (v ? "w" : "-");
      const x = (v) => (v ? "x" : "-");
      return (
        r(p.ownerRead) + w(p.ownerWrite) + x(p.ownerExec) +
        r(p.groupRead) + w(p.groupWrite) + x(p.groupExec) +
        r(p.othersRead) + w(p.othersWrite) + x(p.othersExec)
      );
    },
    onOctalChange() {
      this.parseOctalToPerms(this.octalInput, this.perms);
    },
    onCheckboxChange() {
      this.octalInput = this.permsToOctal(this.perms);
    },
    onDirOctalChange() {
      this.parseOctalToPerms(this.dirOctalInput, this.dirPerms);
    },
    onDirCheckboxChange() {
      this.dirOctalInput = this.permsToOctal(this.dirPerms);
    },
    parseOctalToPerms(input, perms) {
      let val = input.replace(/^0+/, "");
      if (val.length > 3) val = val.slice(-3);
      const num = parseInt(val, 8);
      if (isNaN(num) || num < 0 || num > 0o777) return;

      perms.ownerRead = !!(num & 0o400);
      perms.ownerWrite = !!(num & 0o200);
      perms.ownerExec = !!(num & 0o100);
      perms.groupRead = !!(num & 0o040);
      perms.groupWrite = !!(num & 0o020);
      perms.groupExec = !!(num & 0o010);
      perms.othersRead = !!(num & 0o004);
      perms.othersWrite = !!(num & 0o002);
      perms.othersExec = !!(num & 0o001);
    },
    permsToOctal(perms) {
      let mode = 0;
      if (perms.ownerRead) mode |= 0o400;
      if (perms.ownerWrite) mode |= 0o200;
      if (perms.ownerExec) mode |= 0o100;
      if (perms.groupRead) mode |= 0o040;
      if (perms.groupWrite) mode |= 0o020;
      if (perms.groupExec) mode |= 0o010;
      if (perms.othersRead) mode |= 0o004;
      if (perms.othersWrite) mode |= 0o002;
      if (perms.othersExec) mode |= 0o001;
      return "0" + mode.toString(8).padStart(3, "0");
    },
    async submit() {
      buttons.loading("chmod");

      try {
        const paths = [];
        if (!this.isListing || this.selectedCount === 0) {
          paths.push(this.$route.path);
        } else {
          for (const index of this.selected) {
            paths.push(this.req.items[index].url);
          }
        }

        let modeStr = this.octalInput.replace(/^0+/, "");
        if (modeStr === "") modeStr = "0";
        modeStr = modeStr.padStart(3, "0");

        let dirModeStr = "";
        if (this.recursive) {
          dirModeStr = this.dirOctalInput.replace(/^0+/, "");
          if (dirModeStr === "") dirModeStr = "0";
          dirModeStr = dirModeStr.padStart(3, "0");
        }

        const promises = paths.map((p) =>
          api.chmod(p, modeStr, this.recursive, dirModeStr)
        );
        await Promise.all(promises);

        buttons.success("chmod");
        this.reload = true;
        this.closeHovers();
      } catch (e) {
        buttons.done("chmod");
        this.$showError(e);
      }
    },
  },
};
</script>

<style scoped>
.chmod-modal {
  max-width: 450px;
}

.chmod-section h3 {
  margin: 0.8em 0 0.3em;
  font-size: 0.95em;
  color: #555;
}

.chmod-dir-section {
  border-top: 1px solid #eee;
  padding-top: 0.8em;
  margin-top: 0.8em;
}

.chmod-octal {
  margin-bottom: 0.8em;
}

.chmod-octal label {
  display: block;
  font-weight: bold;
  margin-bottom: 0.3em;
}

.chmod-octal input {
  width: 80px;
  font-family: monospace;
  font-size: 1.1em;
  display: inline-block;
  margin-right: 1em;
}

.chmod-symbolic {
  font-family: monospace;
  font-size: 1.1em;
  color: #666;
}

.chmod-table {
  width: 100%;
  border-collapse: collapse;
  margin: 0.5em 0;
}

.chmod-table th,
.chmod-table td {
  padding: 0.4em 0.8em;
  text-align: center;
}

.chmod-table th:first-child,
.chmod-table td:first-child {
  text-align: left;
}

.chmod-table input[type="checkbox"] {
  width: 18px;
  height: 18px;
  cursor: pointer;
}

.chmod-recursive {
  margin-top: 0.5em;
}

.chmod-recursive label {
  display: flex;
  align-items: center;
  gap: 0.5em;
  cursor: pointer;
}
</style>
